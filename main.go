package main

import (
	"os"
	"context"
	"database/sql"
	"fmt"
	"log"
	"strconv"

	"net/http"

	"dagger.io/dagger"
	"github.com/gin-gonic/gin"

	"github.com/go-sql-driver/mysql"
)

var db *sql.DB

func main() {

	if err := build(context.Background()); err != nil {
		fmt.Println(err)
	}

	cfg := mysql.Config{
		User:                 "powerranger", // os.Getenv("DBUSER"),
		Passwd:               "toorton",     // os.Getenv("DBPASS"),
		Net:                  "tcp",
		Addr:                 "db:3306",
		DBName:               "powerranger",
		AllowNativePasswords: true,
	}

	var err error
	db, err = sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		log.Fatal(err)
	}
	pingErr := db.Ping()
	if pingErr != nil {
		log.Fatal(pingErr)
	}
	fmt.Println("Connected to the Power Ranger database!")

	router := gin.Default()
	router.GET("/pr", getPr)
	router.GET("/pr/id/:id", getPrByID)
	router.GET("/pr/color/:color", getPrByColor)
	router.POST("/pr", postPr)
	router.Run("localhost:8080")
}

type pr struct {
	ID    string  `json:"id"`
	Color string  `json:"color"`
	Name  string  `json:"name"`
	Power float64 `json:"power"`
}

func getPr(c *gin.Context) {
	prs, err := getAllPrDb()
	if prs == nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": err})
	}
	c.IndentedJSON(http.StatusOK, prs)
}

func postPr(c *gin.Context) {
	var newPr pr
	if err := c.BindJSON(&newPr); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": err})
		return
	}

	id, err := addPrDb(newPr)
	if id == 0 {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": err})
		return
	}
	c.IndentedJSON(http.StatusCreated, id)
}

func getPrByID(c *gin.Context) {
	id := c.Param("id")
	floatId, _ := strconv.ParseFloat(id, 32)
	prs, err := getPrByIdDb(floatId)
	if prs != nil {
		c.IndentedJSON(http.StatusOK, prs)
		return
	}
	c.IndentedJSON(http.StatusNotFound, gin.H{"message": err})
}

func getPrByColor(c *gin.Context) {
	color := c.Param("color")
	prs, err := getPrByColorDb(color)
	if prs != nil {
		c.IndentedJSON(http.StatusOK, prs)
		return
	}
	c.IndentedJSON(http.StatusNotFound, gin.H{"message": err})
}

func getAllPrDb() ([]pr, error) {
	var prs []pr
	rows, err := db.Query("SELECT * FROM powerranger")
	if err != nil {
		return nil, fmt.Errorf("getAllPrDb %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var tempPrs pr
		if err := rows.Scan(&tempPrs.ID, &tempPrs.Color, &tempPrs.Name, &tempPrs.Power); err != nil {
			return nil, fmt.Errorf("getAllPrDb %v", err)
		}
		prs = append(prs, tempPrs)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("getAllPrDb %v", err)
	}
	return prs, nil
}

func getPrByIdDb(id float64) ([]pr, error) {
	var prs []pr

	rows, err := db.Query("SELECT * FROM powerranger WHERE id = ?", id)
	if err != nil {
		return nil, fmt.Errorf("prByColor %v: %v", id, err)
	}
	defer rows.Close()
	for rows.Next() {
		var tempPrs pr
		if err := rows.Scan(&tempPrs.ID, &tempPrs.Color, &tempPrs.Name, &tempPrs.Power); err != nil {
			return nil, fmt.Errorf("prByColor %v: %v", id, err)
		}
		prs = append(prs, tempPrs)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("prByColor %v: %v", id, err)
	}
	return prs, nil
}

func getPrByColorDb(color string) ([]pr, error) {
	var prs []pr

	rows, err := db.Query("SELECT * FROM powerranger WHERE color = ?", color)
	if err != nil {
		return nil, fmt.Errorf("prByColor %q: %v", color, err)
	}
	defer rows.Close()
	for rows.Next() {
		var tempPrs pr
		if err := rows.Scan(&tempPrs.ID, &tempPrs.Color, &tempPrs.Name, &tempPrs.Power); err != nil {
			return nil, fmt.Errorf("prByColor %q: %v", color, err)
		}
		prs = append(prs, tempPrs)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("prByColor %q: %v", color, err)
	}
	return prs, nil
}

func addPrDb(pr pr) (int64, error) {
	result, err := db.Exec("INSERT INTO powerranger (color, name, power) VALUES (?, ?, ?)", pr.Color, pr.Name, pr.Power)
	if err != nil {
		return 0, fmt.Errorf("addPr: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("addPr: %v", err)
	}
	return id, nil
}

func build(ctx context.Context) error {
	fmt.Println("Building with Dagger")

	// define build matrix
    oses := []string{"linux", "darwin"}
    arches := []string{"amd64", "arm64"}

	// initialize Dagger client
    client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stdout))
    if err != nil {
        return err
    }
    defer client.Close()

    // get reference to the local project
    src := client.Host().Directory(".")

	// create empty directory to put build outputs
    outputs := client.Directory()

    // get `golang` image
    golang := client.Container().From("golang:latest")

    // mount cloned repository into `golang` image
    golang = golang.WithMountedDirectory("/src", src).WithWorkdir("/src")

    for _, goos := range oses {
        for _, goarch := range arches {
            // create a directory for each os and arch
            path := fmt.Sprintf("build/%s/%s/", goos, goarch)

            // set GOARCH and GOOS in the build environment
            build := golang.WithEnvVariable("GOOS", goos)
            build = build.WithEnvVariable("GOARCH", goarch)

            // build application
            build = build.WithExec([]string{"go", "build", "-o", path})

            // get reference to build output directory in container
            outputs = outputs.WithDirectory(path, build.Directory(path))
        }
    }
    // write build artifacts to host
    _, err = outputs.Export(ctx, ".")
    if err != nil {
        return err
    }

	return nil
}
