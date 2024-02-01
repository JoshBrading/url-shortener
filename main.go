package main

import (
	"database/sql"
	"errors"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

var db *sql.DB

type Redirect struct {
	ID      string
	URL     string
	Clicks  int
	Enabled bool
}

func main() {

	// Load connection string from .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatal("failed to load env", err)
	}

	// Open a connection to DB
	db, err = sql.Open("mysql", os.Getenv("DSN"))
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}

	defer db.Close() // Set db to close when application closes

	if err := db.Ping(); err != nil {
		log.Fatalf("failed to ping: %v", err)
	}

	log.Println("Successfully connected to db")
	router := gin.Default()
	router.GET("/:id", doRedirect)
	router.POST("/create", createRedirect)
	router.Run("localhost:8080")
}

// generateString
// generates a random {length} character alphanumerical string
func generateString(length int) string {

	charset := "aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ123456789"
	randomString := make([]byte, length)

	for i := range randomString {
		randomString[i] = charset[rand.Intn(len(charset))]
	}

	return string(randomString)
}

// createRedirect
// creates a new redirect and posts it to the database
func createRedirect(c *gin.Context) {
	var redirect Redirect
	redirect.ID = generateString(5)
	redirect.Clicks = 0
	if err := c.BindJSON(&redirect); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Insert the redirect into the database
	query := "INSERT INTO redirects (id, url, clicks, enabled) VALUES (?, ?, ?, ?)"
	_, err := db.Exec(query, redirect.ID, redirect.URL, redirect.Clicks, redirect.Enabled)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert redirect into database"})
		return
	}

	// Return created redirect
	c.JSON(http.StatusOK, redirect)
}

// doRedirect
// tells the client to redirect to long url or to 404 on error
func doRedirect(c *gin.Context) {
	var id string = c.Param("id")
	id = strings.ReplaceAll(id, "/", "")
	log.Println(id)
	redirect, err := getRedirectLinkFromId(id)
	if err != nil {
		log.Println(err)
		c.Redirect(http.StatusTemporaryRedirect, "https://beta.rtrn.gg/404")
	}
	c.Redirect(http.StatusTemporaryRedirect, redirect.URL)

	iterateRedirectById(id)
}

func iterateRedirectById(id string) error {
	query := `UPDATE redirects SET clicks = clicks + 1 WHERE id = ?`
	_, err := db.Exec(query, id)
	if err != nil {
		log.Println("Error updating click count:", err)
		return err
	}
	return nil
}

// getRedirectLinkFromId
// returns a redirect object from the database containing id, url, clicks, and enabled
func getRedirectLinkFromId(id string) (Redirect, error) {
	var redirect Redirect
	query := `SELECT * FROM redirects WHERE id = ?`

	// Search for row and fill redirect with response
	err := db.QueryRow(query, id).Scan(&redirect.ID, &redirect.URL, &redirect.Clicks, &redirect.Enabled)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Println("No row found with that ID")
			return redirect, errors.New("No row found with that ID")
		}
		log.Println("Error fetching redirect:", err)
		return redirect, err
	}
	return redirect, nil
}
