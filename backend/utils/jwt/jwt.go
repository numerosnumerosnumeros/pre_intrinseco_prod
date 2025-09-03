package jwt

import (
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"sync"
	"time"

	"nodofinance/utils/csrf"
	"nodofinance/utils/env"
	"nodofinance/utils/logger"

	"github.com/golang-jwt/jwt/v5"
	"github.com/patrickmn/go-cache"
)

type JWKS struct {
	Keys []json.RawMessage `json:"keys"`
}

var (
	jwksCache *JWKS
	cacheLock sync.RWMutex
)

var CLIENT_ID string
var USER_POOL_ID string
var ISS_CLAIM string

var validationCache *cache.Cache

func LoadJWKS() error {
	if validationCache == nil {
		validationCache = cache.New(2*time.Minute, 5*time.Minute)
		logger.Log.Info("JWT token validation cache initialized")
	}

	if CLIENT_ID == "" {
		clientID, exists := env.Get("COGNITO_CLIENT_ID")
		if !exists {
			logger.Log.Fatal("Failed to load CLIENT_ID")
		}
		CLIENT_ID = clientID
	}

	if USER_POOL_ID == "" {
		issClaim, exists := env.Get("COGNITO_USER_POOL_ID")
		if !exists {
			logger.Log.Fatal("Failed to load USER_POOL_ID")
		}
		USER_POOL_ID = issClaim
	}

	if ISS_CLAIM == "" {
		ISS_CLAIM = fmt.Sprintf("https://cognito-idp.%s.amazonaws.com/%s", os.Getenv("AWS_REGION"), USER_POOL_ID)
	}

	const (
		maxRetries = 3
		retryDelay = 2 * time.Second
	)

	jwksURL, exists := env.Get("COGNITO_TOKEN_SIGNING_KEY_URL")
	if !exists {
		logger.Log.Fatal("Failed to load Cognito token signing key")
	}

	cacheLock.Lock()
	defer cacheLock.Unlock()

	var err error
	for range maxRetries {
		resp, err := http.Get(jwksURL)
		if err != nil {
			time.Sleep(retryDelay)
			continue
		}

		defer resp.Body.Close()

		var jwks JWKS
		if err = json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
			time.Sleep(retryDelay)
			continue
		}
		jwksCache = &jwks
		return nil // Success
	}

	return err
}

func ValidateToken(tokenString string, needsPremiumValidation bool) (bool, error) {
	hasher := sha256.New()
	hasher.Write([]byte(tokenString))
	cacheKey := hex.EncodeToString(hasher.Sum(nil)) + ":" + fmt.Sprintf("%v", needsPremiumValidation)

	if cachedResult, found := validationCache.Get(cacheKey); found {
		return cachedResult.(bool), nil
	}

	firstTry := true
	for range 2 {

		cacheLock.RLock()
		localCache := jwksCache
		cacheLock.RUnlock()

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}

			kid, ok := token.Header["kid"].(string)
			if !ok {
				return nil, fmt.Errorf("expecting jwt header to have string 'kid'")
			}

			for _, key := range localCache.Keys {
				var keyData struct {
					Kid string `json:"kid"`
					N   string `json:"n"`
					E   string `json:"e"`
					Kty string `json:"kty"`
					Alg string `json:"alg"`
					Use string `json:"use"`
				}
				if err := json.Unmarshal(key, &keyData); err != nil {
					continue
				}

				if kid == keyData.Kid {
					// Decode base64 URL-encoded values of N and E to construct RSA public key
					nBytes, err := base64.RawURLEncoding.DecodeString(keyData.N)
					if err != nil {
						continue
					}
					eBytes, err := base64.RawURLEncoding.DecodeString(keyData.E)
					if err != nil {
						continue
					}

					n := new(big.Int).SetBytes(nBytes)
					e := big.NewInt(0).SetBytes(eBytes).Int64()

					pubKey := &rsa.PublicKey{N: n, E: int(e)}
					return pubKey, nil
				}
			}
			return nil, fmt.Errorf("key with kid %s not found in jwks in utils.validatetoken", kid)
		})

		if err != nil {
			if firstTry {
				firstTry = false
				// handle jwks rotation
				LoadJWKS()
				continue
			}

			return false, nil
		}

		firstTry = false

		if !token.Valid {
			return false, nil
		}

		// Check expiration
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			if exp, ok := claims["exp"].(float64); ok {
				if time.Now().Unix() > int64(exp) {
					return false, nil
				}
			}
		}

		// Check token_use
		var tokenUse string
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			if use, ok := claims["token_use"].(string); ok {
				tokenUse = use
			}
		}
		// Token-specific validations
		if tokenUse == "id" {
			if claims, ok := token.Claims.(jwt.MapClaims); ok {
				if aud, ok := claims["aud"].(string); ok {
					if aud != CLIENT_ID {
						return false, nil
					}
				}

				emailVerifiedClaim, exists := claims["email_verified"].(bool)
				if !exists {
					return false, nil
				}
				if !emailVerifiedClaim {
					return false, nil
				}

				// Premium validation
				if needsPremiumValidation {
					subscriptionStatusClaim, exists := claims["custom:subscription_status"].(string)
					if !exists {
						return false, nil
					}
					if subscriptionStatusClaim != "active" {
						return false, nil
					}
				}
			}
		} else if tokenUse == "access" {
			if claims, ok := token.Claims.(jwt.MapClaims); ok {
				if clientID, ok := claims["client_id"].(string); ok {
					if clientID != CLIENT_ID {
						return false, nil
					}
				}
			}
		} else {
			return false, nil
		}

		// Issuer claim
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			if iss, ok := claims["iss"].(string); ok {
				if iss != ISS_CLAIM {
					return false, nil
				}
			}
		}

		validationCache.Set(cacheKey, true, cache.DefaultExpiration)

		return true, nil
	}

	return false, nil
}

func GetTokenClaims(tokenPassed string) (jwt.MapClaims, error) {
	if tokenPassed == "" {
		return nil, nil
	}

	token, _, err := jwt.NewParser().ParseUnverified(tokenPassed, jwt.MapClaims{})
	if err != nil {
		return nil, fmt.Errorf("could not parse token in utils.gettokenclaims: %w", err)
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		return claims, nil
	}

	return nil, fmt.Errorf("could not assert claims type in utils.gettokenclaims")
}

func ValidateRequest(r *http.Request, needPremiumValidation bool) bool {
	accessTokenCookie, errAT := r.Cookie("nodo_access_token")
	idTokenCookie, errIT := r.Cookie("nodo_id_token")
	if errAT != nil {
		return false
	}
	if errIT != nil {
		return false
	}

	accessTokenValidChan := make(chan bool, 1)
	accessTokenErrorChan := make(chan error, 1)
	idTokenValidChan := make(chan bool, 1)
	idTokenErrorChan := make(chan error, 1)
	csrfValidChan := make(chan bool, 1)

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		isValid, err := ValidateToken(accessTokenCookie.Value, false)
		accessTokenValidChan <- isValid
		accessTokenErrorChan <- err
	}()

	go func() {
		defer wg.Done()
		isValid, err := ValidateToken(idTokenCookie.Value, needPremiumValidation)
		idTokenValidChan <- isValid
		idTokenErrorChan <- err
	}()

	go func() {
		defer wg.Done()
		isValid := csrf.Validate(r)
		csrfValidChan <- isValid
	}()

	wg.Wait()

	isValidAccessToken := <-accessTokenValidChan
	errAT = <-accessTokenErrorChan
	isValidIdToken := <-idTokenValidChan
	errIT = <-idTokenErrorChan
	isValidCSRF := <-csrfValidChan

	close(accessTokenValidChan)
	close(accessTokenErrorChan)
	close(idTokenValidChan)
	close(idTokenErrorChan)
	close(csrfValidChan)

	if errAT != nil || errIT != nil {
		return false
	}

	if !isValidCSRF || !isValidAccessToken || !isValidIdToken {
		return false
	}

	return true
}
