package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/JunoAX/housepoints-go/internal/auth"
	"github.com/JunoAX/housepoints-go/internal/middleware"
	"github.com/JunoAX/housepoints-go/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token    string    `json:"token"`
	UserID   uuid.UUID `json:"user_id"`
	Username string    `json:"username"`
	IsParent bool      `json:"is_parent"`
	FamilyID uuid.UUID `json:"family_id"`
}

// Login authenticates a user and returns a JWT token
func Login(jwtService *auth.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		db, ok := middleware.GetFamilyDB(c)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not found"})
			return
		}

		family, ok := middleware.GetFamily(c)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Family context required"})
			return
		}

		var req LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
			return
		}

		// Normalize username to lowercase
		username := strings.ToLower(strings.TrimSpace(req.Username))

		// Query user from family database
		query := `
			SELECT id, username, password_hash, is_parent, login_enabled
			FROM users
			WHERE LOWER(username) = $1
		`

		var userID uuid.UUID
		var dbUsername string
		var passwordHash *string
		var isParent bool
		var loginEnabled bool

		err := db.QueryRow(c.Request.Context(), query, username).Scan(
			&userID, &dbUsername, &passwordHash, &isParent, &loginEnabled,
		)

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
			return
		}

		// Check if login is enabled
		if !loginEnabled {
			c.JSON(http.StatusForbidden, gin.H{"error": "Login is disabled for this user"})
			return
		}

		// Check if password_hash exists
		if passwordHash == nil || *passwordHash == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Password authentication not configured for this user"})
			return
		}

		// Verify password
		err = bcrypt.CompareHashAndPassword([]byte(*passwordHash), []byte(req.Password))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
			return
		}

		// Generate JWT token
		token, err := jwtService.GenerateToken(userID, family.ID, dbUsername, isParent)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
			return
		}

		// Return token and user info
		c.JSON(http.StatusOK, LoginResponse{
			Token:    token,
			UserID:   userID,
			Username: dbUsername,
			IsParent: isParent,
			FamilyID: family.ID,
		})
	}
}

// Google OAuth configuration
var (
	GoogleWebClientID = "1005514268333-l3oivohqi45ig05pegqovvh0dd23r2f2.apps.googleusercontent.com"
	GoogleIOSClientID = "1005514268333-c44cq90gh92sek2hjg9jjf68b53ear76.apps.googleusercontent.com"

	// Authorized family emails (case-insensitive)
	AuthorizedEmails = map[string]struct {
		Username string
		IsParent bool
	}{
		"tom.gamull@gmail.com":      {Username: "tom", IsParent: true},
		"isabellamck92@gmail.com":   {Username: "isabella", IsParent: true},
		"gus.gamull@gmail.com":      {Username: "gus", IsParent: false},
		"gamullmo@gmail.com":        {Username: "mo", IsParent: false},
		"gamull.mo@gmail.com":       {Username: "mo", IsParent: false},
		"james.gamull@gmail.com":    {Username: "james", IsParent: false},
		"john.gamull@gmail.com":     {Username: "john", IsParent: false},
		"john.gamull.com":           {Username: "john", IsParent: false},
	}
)

type GoogleMobileAuthRequest struct {
	IDToken string `json:"id_token"`
	IdToken string `json:"idToken"` // Accept camelCase too
}

type GoogleTokenInfo struct {
	Aud           string `json:"aud"`
	Email         string `json:"email"`
	EmailVerified string `json:"email_verified"`
	Name          string `json:"name"`
	Sub           string `json:"sub"`
}

type GoogleMobileAuthResponse struct {
	Token string `json:"token"`
	User  struct {
		ID          string `json:"id"`
		Username    string `json:"username"`
		DisplayName string `json:"display_name"`
		Email       string `json:"email"`
		IsParent    bool   `json:"is_parent"`
		IsChild     bool   `json:"is_child"`
	} `json:"user"`
}

// GoogleMobileAuth handles Google OAuth for mobile app
func GoogleMobileAuth(jwtService *auth.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		db, ok := middleware.GetFamilyDB(c)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not found"})
			return
		}

		family, ok := middleware.GetFamily(c)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Family context required"})
			return
		}

		var req GoogleMobileAuthRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
			return
		}

		// Get ID token (support both snake_case and camelCase)
		idToken := req.IDToken
		if idToken == "" {
			idToken = req.IdToken
		}
		if idToken == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing id_token"})
			return
		}

		// Verify token with Google
		verifyURL := fmt.Sprintf("https://oauth2.googleapis.com/tokeninfo?id_token=%s", idToken)
		resp, err := http.Get(verifyURL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify token"})
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token", "details": string(body)})
			return
		}

		// Parse token info
		var tokenInfo GoogleTokenInfo
		if err := json.NewDecoder(resp.Body).Decode(&tokenInfo); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse token info"})
			return
		}

		// Verify client ID (accept both web and iOS)
		if tokenInfo.Aud != GoogleWebClientID && tokenInfo.Aud != GoogleIOSClientID {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid client ID"})
			return
		}

		// Check if user is authorized
		email := strings.ToLower(tokenInfo.Email)
		userConfig, authorized := AuthorizedEmails[email]
		if !authorized {
			c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized user"})
			return
		}

		username := userConfig.Username
		isParent := userConfig.IsParent
		displayName := tokenInfo.Name
		if displayName == "" {
			displayName = username
		}

		// Get or create user in database
		ctx := context.Background()
		var userID uuid.UUID
		now := time.Now()

		// Check if user exists
		query := `SELECT id FROM users WHERE username = $1`
		err = db.QueryRow(ctx, query, username).Scan(&userID)

		if err != nil {
			// User doesn't exist, create new one
			userID = uuid.New()
			insertQuery := `
				INSERT INTO users (
					id, username, display_name, email, is_parent,
					total_points, weekly_points, level, xp, streak_days,
					last_login, last_active, created_at, updated_at
				) VALUES ($1, $2, $3, $4, $5, 0, 0, 1, 0, 0, $6, $7, $8, $9)
			`
			_, err = db.Exec(ctx, insertQuery,
				userID, username, displayName, email, isParent,
				now, now, now, now,
			)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user", "details": err.Error()})
				return
			}
		} else {
			// User exists, update email and last_login
			updateQuery := `
				UPDATE users
				SET email = $1, last_login = $2, last_active = $3, updated_at = $4
				WHERE id = $5
			`
			_, err = db.Exec(ctx, updateQuery, email, now, now, now, userID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user", "details": err.Error()})
				return
			}
		}

		// Log authentication event (optional, best effort)
		go func() {
			logQuery := `
				INSERT INTO auth_logs (user_id, event_type, details, ip_address, user_agent)
				VALUES ($1, $2, $3, $4, $5)
			`
			details := fmt.Sprintf(`{"method":"google_mobile","email":"%s"}`, email)
			db.Exec(context.Background(), logQuery,
				userID, "login", details, "mobile_app", "mobile",
			)
		}()

		// Generate JWT token
		token, err := jwtService.GenerateToken(userID, family.ID, username, isParent)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
			return
		}

		// Return response
		response := GoogleMobileAuthResponse{
			Token: token,
		}
		response.User.ID = userID.String()
		response.User.Username = username
		response.User.DisplayName = displayName
		response.User.Email = email
		response.User.IsParent = isParent
		response.User.IsChild = !isParent

		c.JSON(http.StatusOK, response)
	}
}

// OAuth state management (in-memory, for production use Redis)
var (
	oauthStates     = make(map[string]OAuthState)
	oauthStatesMux  sync.RWMutex
	stateCleanupDone = false
)

type OAuthState struct {
	FamilySlug   string
	RedirectPath string
	CreatedAt    time.Time
}

func init() {
	// Start cleanup goroutine once
	if !stateCleanupDone {
		go cleanupExpiredStates()
		stateCleanupDone = true
	}
}

func cleanupExpiredStates() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		oauthStatesMux.Lock()
		now := time.Now()
		for state, data := range oauthStates {
			if now.Sub(data.CreatedAt) > 10*time.Minute {
				delete(oauthStates, state)
			}
		}
		oauthStatesMux.Unlock()
	}
}

func generateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// GoogleWebInit initializes Google OAuth flow for web
func GoogleWebInit() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get family from middleware (optional for init)
		var familySlug string
		if family, exists := middleware.GetFamily(c); exists {
			familySlug = family.Slug
		} else {
			// Extract from subdomain
			host := c.Request.Host
			parts := strings.Split(host, ".")
			if len(parts) > 0 && parts[0] != "www" {
				familySlug = parts[0]
			}
		}

		redirectPath := c.Query("redirect_uri")
		if redirectPath == "" {
			redirectPath = "/dashboard"
		}

		// Generate CSRF state token
		state, err := generateState()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate state"})
			return
		}

		// Store state
		oauthStatesMux.Lock()
		oauthStates[state] = OAuthState{
			FamilySlug:   familySlug,
			RedirectPath: redirectPath,
			CreatedAt:    time.Now(),
		}
		oauthStatesMux.Unlock()

		// Get Google OAuth config from environment
		clientID := os.Getenv("GOOGLE_CLIENT_ID")
		if clientID == "" {
			clientID = GoogleWebClientID
		}

		callbackURL := os.Getenv("GOOGLE_REDIRECT_URI")
		if callbackURL == "" {
			callbackURL = "https://housepoints.ai/api/auth/google/callback"
		}

		// Build Google OAuth URL
		params := url.Values{}
		params.Add("client_id", clientID)
		params.Add("redirect_uri", callbackURL)
		params.Add("response_type", "code")
		params.Add("scope", "openid email profile")
		params.Add("state", state)
		params.Add("access_type", "offline")
		params.Add("prompt", "select_account")

		authURL := "https://accounts.google.com/o/oauth2/v2/auth?" + params.Encode()

		// Return redirect URL for frontend to use
		c.JSON(http.StatusOK, gin.H{
			"authorization_url": authURL,
		})
	}
}

// GoogleWebCallback handles Google OAuth callback
func GoogleWebCallback(jwtService *auth.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		code := c.Query("code")
		state := c.Query("state")

		if code == "" || state == "" {
			c.Redirect(http.StatusTemporaryRedirect, "/?error=missing_parameters")
			return
		}

		// Verify and get state
		oauthStatesMux.RLock()
		stateData, exists := oauthStates[state]
		oauthStatesMux.RUnlock()

		if !exists {
			c.Redirect(http.StatusTemporaryRedirect, "/?error=invalid_state")
			return
		}

		// Remove used state
		oauthStatesMux.Lock()
		delete(oauthStates, state)
		oauthStatesMux.Unlock()

		// Exchange code for tokens
		clientID := os.Getenv("GOOGLE_CLIENT_ID")
		clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
		callbackURL := os.Getenv("GOOGLE_REDIRECT_URI")

		if clientID == "" {
			clientID = GoogleWebClientID
		}
		if callbackURL == "" {
			callbackURL = "https://housepoints.ai/api/auth/google/callback"
		}

		tokenURL := "https://oauth2.googleapis.com/token"
		data := url.Values{}
		data.Set("code", code)
		data.Set("client_id", clientID)
		data.Set("client_secret", clientSecret)
		data.Set("redirect_uri", callbackURL)
		data.Set("grant_type", "authorization_code")

		resp, err := http.PostForm(tokenURL, data)
		if err != nil {
			c.Redirect(http.StatusTemporaryRedirect, "/?error=token_exchange_failed")
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			c.Redirect(http.StatusTemporaryRedirect, "/?error=token_exchange_failed")
			return
		}

		var tokenResp struct {
			IDToken string `json:"id_token"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
			c.Redirect(http.StatusTemporaryRedirect, "/?error=token_parse_failed")
			return
		}

		// Verify ID token
		verifyURL := fmt.Sprintf("https://oauth2.googleapis.com/tokeninfo?id_token=%s", tokenResp.IDToken)
		verifyResp, err := http.Get(verifyURL)
		if err != nil {
			c.Redirect(http.StatusTemporaryRedirect, "/?error=token_verify_failed")
			return
		}
		defer verifyResp.Body.Close()

		if verifyResp.StatusCode != http.StatusOK {
			c.Redirect(http.StatusTemporaryRedirect, "/?error=invalid_token")
			return
		}

		var tokenInfo GoogleTokenInfo
		if err := json.NewDecoder(verifyResp.Body).Decode(&tokenInfo); err != nil {
			c.Redirect(http.StatusTemporaryRedirect, "/?error=token_info_parse_failed")
			return
		}

		// Verify client ID
		if tokenInfo.Aud != GoogleWebClientID && tokenInfo.Aud != GoogleIOSClientID {
			c.Redirect(http.StatusTemporaryRedirect, "/?error=invalid_client")
			return
		}

		// Get family database using platform DB
		familySlug := stateData.FamilySlug
		if familySlug == "" {
			c.Redirect(http.StatusTemporaryRedirect, "/?error=missing_family")
			return
		}

		// Check authorization based on family
		email := strings.ToLower(tokenInfo.Email)
		var username string
		var isParent bool

		if familySlug == "demo" {
			// For demo family, check demo_access_requests table
			// This will be checked when connecting to platform DB below
			username = strings.Split(email, "@")[0] // Use email prefix as username
			isParent = false                         // Demo users are not parents by default
		} else {
			// For production families, check hardcoded authorized emails
			userConfig, authorized := AuthorizedEmails[email]
			if !authorized {
				c.Redirect(http.StatusTemporaryRedirect, "/?error=unauthorized_email")
				return
			}
			username = userConfig.Username
			isParent = userConfig.IsParent
		}

		// Connect to platform DB to get family info
		platformDBURL := os.Getenv("PLATFORM_DATABASE_URL")
		if platformDBURL == "" {
			// Build from individual env vars
			dbHost := os.Getenv("DB_HOST")
			dbPort := os.Getenv("DB_PORT")
			dbUser := os.Getenv("DB_USER")
			dbPassword := os.Getenv("DB_PASSWORD")
			platformDBName := os.Getenv("PLATFORM_DB_NAME")

			// Use defaults if not set
			if dbHost == "" {
				dbHost = "10.1.10.20"
			}
			if dbPort == "" {
				dbPort = "5432"
			}
			if dbUser == "" {
				dbUser = "postgres"
			}
			if dbPassword == "" {
				dbPassword = "HP_Sec2025_O0mZVY90R1Yg8L"
			}
			if platformDBName == "" {
				platformDBName = "housepoints_platform"
			}

			platformDBURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
				dbUser, dbPassword, dbHost, dbPort, platformDBName)
		}

		ctx := c.Request.Context()
		platformPool, err := pgxpool.New(ctx, platformDBURL)
		if err != nil {
			log.Printf("Failed to connect to platform database: %v", err)
			c.Redirect(http.StatusTemporaryRedirect, "/?error=database_connection_failed")
			return
		}
		defer platformPool.Close()

		// Query for family
		var family models.Family
		err = platformPool.QueryRow(ctx, `
			SELECT id, slug, name, db_host, db_port, db_name, db_user, db_password_encrypted,
			       plan, status, created_at, updated_at
			FROM families
			WHERE slug = $1 AND status = 'active' AND deleted_at IS NULL
		`, familySlug).Scan(
			&family.ID, &family.Slug, &family.Name,
			&family.DBHost, &family.DBPort, &family.DBName, &family.DBUser, &family.DBPasswordEncrypted,
			&family.Plan, &family.Status, &family.CreatedAt, &family.UpdatedAt,
		)
		if err != nil {
			log.Printf("Family not found or inactive: %s - %v", familySlug, err)
			c.Redirect(http.StatusTemporaryRedirect, "/?error=family_not_found")
			return
		}

		// For demo family, check demo access approval
		if familySlug == "demo" {
			var accessStatus string
			err = platformPool.QueryRow(ctx, `
				SELECT status FROM demo_access_requests
				WHERE LOWER(email) = LOWER($1) AND status = 'approved'
			`, email).Scan(&accessStatus)

			if err != nil {
				log.Printf("Demo access not found or not approved for email: %s", email)
				// Auto-approve by inserting into demo_access_requests
				_, insertErr := platformPool.Exec(ctx, `
					INSERT INTO demo_access_requests (email, google_email, status, approved_at)
					VALUES ($1, $2, 'approved', NOW())
					ON CONFLICT (email) DO NOTHING
				`, email, email)

				if insertErr != nil {
					log.Printf("Failed to auto-approve demo access: %v", insertErr)
					c.Redirect(http.StatusTemporaryRedirect, "/demo/request-access?email="+url.QueryEscape(email))
					return
				}
				log.Printf("Auto-approved demo access for: %s", email)
			}
		}

		// Build family database connection string
		familyDBURL := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
			family.DBUser, family.DBPasswordEncrypted, family.DBHost, family.DBPort, family.DBName)

		// Connect to family database
		familyPool, err := pgxpool.New(ctx, familyDBURL)
		if err != nil {
			log.Printf("Failed to connect to family database %s: %v", family.DBName, err)
			c.Redirect(http.StatusTemporaryRedirect, "/?error=family_database_failed")
			return
		}
		defer familyPool.Close()

		// Check if user exists
		var userID uuid.UUID
		var existingIsParent bool
		err = familyPool.QueryRow(ctx, `
			SELECT id, is_parent FROM users WHERE username = $1
		`, username).Scan(&userID, &existingIsParent)

		if err != nil {
			// User doesn't exist, create them
			userID = uuid.New()

			_, err = familyPool.Exec(ctx, `
				INSERT INTO users (id, username, email, is_parent, family_id, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
			`, userID, username, email, isParent, family.ID)

			if err != nil {
				log.Printf("Failed to create user %s in family %s: %v", username, family.Slug, err)
				c.Redirect(http.StatusTemporaryRedirect, "/?error=user_creation_failed")
				return
			}

			log.Printf("Created new user %s (%s) in family %s", username, userID, family.Slug)
		} else {
			// User exists, update email if changed
			_, err = familyPool.Exec(ctx, `
				UPDATE users SET email = $1, updated_at = NOW() WHERE id = $2
			`, email, userID)

			if err != nil {
				log.Printf("Failed to update user %s: %v", username, err)
				// Continue anyway, this is non-critical
			}

			log.Printf("User %s (%s) logged in to family %s", username, userID, family.Slug)
		}

		// Generate JWT token
		token, err := jwtService.GenerateToken(userID, family.ID, username, isParent)
		if err != nil {
			log.Printf("Failed to generate JWT token for user %s: %v", username, err)
			c.Redirect(http.StatusTemporaryRedirect, "/?error=token_generation_failed")
			return
		}

		// Parse redirect path - handle both full URLs and paths
		redirectPath := stateData.RedirectPath
		if redirectPath == "" {
			redirectPath = "/dashboard"
		}

		// If redirectPath contains a full URL, extract just the path
		if strings.HasPrefix(redirectPath, "http://") || strings.HasPrefix(redirectPath, "https://") {
			parsedURL, err := url.Parse(redirectPath)
			if err == nil {
				redirectPath = parsedURL.Path
				if parsedURL.RawQuery != "" {
					redirectPath += "?" + parsedURL.RawQuery
				}
			}
		}

		// Redirect to family subdomain with JWT token
		redirectURL := fmt.Sprintf("https://%s.housepoints.ai%s?token=%s&google_login=true",
			familySlug, redirectPath, url.QueryEscape(token))

		log.Printf("OAuth callback complete for %s@%s, redirecting to %s", username, familySlug, redirectURL)
		c.Redirect(http.StatusTemporaryRedirect, redirectURL)
	}
}
