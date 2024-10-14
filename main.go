package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

var db *sql.DB
var driver string

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	log.Printf("Environment variable %s not set, using default value: %s", key, defaultValue)
	return defaultValue
}

func initDB() {
	log.Println("Initializing database connection...")
	var err error
	username := getEnv("DB_USERNAME", "root")
	password := getEnv("DB_PASSWORD", "")
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "3306")
	database := getEnv("DB_DATABASE", "db_wilayah")
	driver = getEnv("DB_DRIVER", "mysql")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", username, password, host, port, database)
	if driver == "postgres" {
		dsn = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=require&binary_parameters=yes", username, password, host, port, database)
	}

	log.Printf("Attempting to connect to %s database at %s:%s...", driver, host, port)
	db, err = sql.Open(driver, dsn)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}

	if err = db.Ping(); err != nil {
		log.Fatalf("Error pinging database: %v", err)
	}

	log.Println("Successfully connected to the database")
}

func checkTableExist() {
	log.Println("Checking if required tables exist...")
	tables := []string{"provinsis", "kab_kotas", "kecamatans", "kelurahan_desas"}
	for _, table := range tables {
		log.Printf("Checking table: %s", table)
		if _, err := db.Query(fmt.Sprintf("SELECT 1 FROM %s LIMIT 1", table)); err != nil {
			log.Fatalf("Table %s does not exist: %v", table, err)
		}
		log.Printf("Table %s exists", table)
	}
	log.Println("All required tables exist")
}

func info(c *gin.Context) {
	log.Println("Handling request for info endpoint")
	counts := make(map[string]int)
	tables := []string{"provinsis", "kab_kotas", "kecamatans", "kelurahan_desas"}

	for _, table := range tables {
		var count int
		log.Printf("Counting records in %s table", table)
		if err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count); err != nil {
			log.Printf("Error counting %s: %v", table, err)
			c.JSON(500, gin.H{"error": fmt.Sprintf("Error counting %s", table)})
			return
		}
		counts[table] = count
		log.Printf("%s count: %d", table, count)
	}

	c.JSON(200, gin.H{
		"jumlah_provinsi":  counts["provinsis"],
		"jumlah_kabupaten": counts["kab_kotas"],
		"jumlah_kecamatan": counts["kecamatans"],
		"jumlah_kelurahan": counts["kelurahan_desas"],
	})
	log.Println("Info request handled successfully")
}

func getItems(c *gin.Context, tableName, columnName string) {
	log.Printf("Handling request for %s items", tableName)
	searchQuery := c.Query("search")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	log.Printf("Search query: %s, Page: %d, Limit: %d", searchQuery, page, limit)

	query := fmt.Sprintf("SELECT id, %s FROM %s", columnName, tableName)
	args := make([]interface{}, 0)

	if searchQuery != "" {
		query += fmt.Sprintf(" WHERE %s LIKE ?", columnName)
		args = append(args, "%"+searchQuery+"%")
	}

	query += " ORDER BY id ASC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	if driver == "postgres" {
		query = convertToPostgres(query)
	}

	log.Printf("Executing query: %s", query)
	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("Error executing query: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var items []map[string]interface{}
	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}
		items = append(items, map[string]interface{}{"id": id, "nama": name})
	}

	log.Printf("Retrieved %d items from %s", len(items), tableName)
	c.JSON(200, items)
}

func getProvinsi(c *gin.Context) {
	log.Println("Handling request for provinces")
	getItems(c, "provinsis", "nama_provinsi")
}

func getKabupatenAll(c *gin.Context) {
	log.Println("Handling request for all kabupaten")
	getItems(c, "kab_kotas", "nama_kab_kota")
}

func getKecamatanAll(c *gin.Context) {
	log.Println("Handling request for all kecamatan")
	getItems(c, "kecamatans", "nama_kecamatan")
}

func getKelurahanAll(c *gin.Context) {
	log.Println("Handling request for all kelurahan")
	getItems(c, "kelurahan_desas", "nama_kelurahan_desa")
}

func getDetailItem(c *gin.Context, tableName, columnName string, additionalCounts ...string) {
	id := c.Param("id")
	log.Printf("Handling request for %s detail with id: %s", tableName, id)

	query := fmt.Sprintf("SELECT id, %s FROM %s WHERE id = ?", columnName, tableName)
	if driver == "postgres" {
		query = convertToPostgres(query)
	}

	log.Printf("Executing query: %s", query)
	var itemID int
	var itemName string
	if err := db.QueryRow(query, id).Scan(&itemID, &itemName); err != nil {
		log.Printf("Error retrieving %s detail: %v", tableName, err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	result := map[string]interface{}{
		"id":   itemID,
		"nama": itemName,
	}

	for _, countTable := range additionalCounts {
		var count int
		countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s_id = ?", countTable, tableName[:len(tableName)-1])
		if driver == "postgres" {
			countQuery = convertToPostgres(countQuery)
		}
		log.Printf("Executing count query: %s", countQuery)
		if err := db.QueryRow(countQuery, id).Scan(&count); err != nil {
			log.Printf("Error counting %s: %v", countTable, err)
			continue
		}
		result[fmt.Sprintf("jumlah_%s", countTable)] = count
		log.Printf("Count for %s: %d", countTable, count)
	}

	log.Printf("Retrieved detail for %s with id: %s", tableName, id)
	c.JSON(200, result)
}

func getDetailProvinsi(c *gin.Context) {
	log.Println("Handling request for province detail")
	getDetailItem(c, "provinsis", "nama_provinsi", "kab_kotas", "kecamatans", "kelurahan_desas")
}

func getDetailKabupaten(c *gin.Context) {
	log.Println("Handling request for kabupaten detail")
	getDetailItem(c, "kab_kotas", "nama_kab_kota", "kecamatans", "kelurahan_desas")
}

func getDetailKecamatan(c *gin.Context) {
	log.Println("Handling request for kecamatan detail")
	getDetailItem(c, "kecamatans", "nama_kecamatan", "kelurahan_desas")
}

func getDetailKelurahan(c *gin.Context) {
	log.Println("Handling request for kelurahan detail")
	getDetailItem(c, "kelurahan_desas", "nama_kelurahan_desa")
}

func getChildItems(c *gin.Context, parentTable, childTable, parentColumn, childColumn string) {
	parentID := c.Param("id")
	log.Printf("Handling request for %s of %s with id: %s", childTable, parentTable, parentID)

	query := fmt.Sprintf("SELECT id, %s FROM %s WHERE %s = ?", childColumn, childTable, parentColumn)
	if driver == "postgres" {
		query = convertToPostgres(query)
	}

	log.Printf("Executing query: %s", query)
	rows, err := db.Query(query, parentID)
	if err != nil {
		log.Printf("Error querying %s: %v", childTable, err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var items []map[string]interface{}
	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}
		items = append(items, map[string]interface{}{"id": id, "nama": name})
	}

	log.Printf("Retrieved %d %s for %s with id: %s", len(items), childTable, parentTable, parentID)
	c.JSON(200, items)
}

func getKabupaten(c *gin.Context) {
	log.Println("Handling request for kabupaten by province")
	getChildItems(c, "provinsis", "kab_kotas", "provinsi_id", "nama_kab_kota")
}

func getKecamatan(c *gin.Context) {
	log.Println("Handling request for kecamatan by kabupaten")
	getChildItems(c, "kab_kotas", "kecamatans", "kab_kota_id", "nama_kecamatan")
}

func getKelurahan(c *gin.Context) {
	log.Println("Handling request for kelurahan by kecamatan")
	getChildItems(c, "kecamatans", "kelurahan_desas", "kecamatan_id", "nama_kelurahan_desa")
}

func convertToPostgres(query string) string {
	log.Println("Converting MySQL query to PostgreSQL format")
	paramCount := 1
	for i := 0; i < len(query); i++ {
		if query[i] == '?' {
			query = query[:i] + fmt.Sprintf("$%d", paramCount) + query[i+1:]
			paramCount++
		}
	}
	log.Printf("Converted query: %s", query)
	return query
}

func main() {
	log.Println("Starting application...")
	if err := godotenv.Load(".env"); err != nil {
		log.Println("No .env file found, using default environment variables")
	} else {
		log.Println("Loaded environment variables from .env file")
	}

	port := getEnv("PORT", "8080")
	log.Printf("Using port: %s", port)

	initDB()
	checkTableExist()
	defer db.Close()

	log.Println("Setting up Gin router...")
	router := gin.Default()

	router.GET("/", info)
	router.GET("/provinsi", getProvinsi)
	router.GET("/provinsi/:id", getDetailProvinsi)
	router.GET("/kota", getKabupatenAll)
	router.GET("/provinsi/:id/kota", getKabupaten)
	router.GET("/kota/:id", getDetailKabupaten)
	router.GET("/kecamatan", getKecamatanAll)
	router.GET("/kota/:id/kecamatan", getKecamatan)
	router.GET("/kecamatan/:id", getDetailKecamatan)
	router.GET("/kelurahan", getKelurahanAll)
	router.GET("/kecamatan/:id/kelurahan", getKelurahan)
	router.GET("/kelurahan/:id", getDetailKelurahan)

	log.Printf("Starting server on 0.0.0.0:%s", port)
	router.Run("0.0.0.0:" + port)
}
