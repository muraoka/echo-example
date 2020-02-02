package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo"
	_ "github.com/lib/pq"
	"gopkg.in/gorp.v2"
)

var dbDriver = "postgres"

type Comment struct {
	Id      int64     `json:"id" db:"id,primarykey,autoincrement"`
	Name    string    `json:"name" db:"name,notnull,size:200"`
	Text    string    `json:"text"  db:"text,notnull,size:399"`
	Created time.Time `json:"created" db:"created,notnull"`
	Updated time.Time `json:"updated" db:"updated,notnull"`
}

func setupDB() (*gorp.DbMap, error) {
	db, err := sql.Open(dbDriver, os.Getenv("DSN"))
	if err != nil {
		return nil, err
	}

	var diarect gorp.Dialect = gorp.PostgresDialect{}

	dbmap := &gorp.DbMap{Db: db, Dialect: diarect}
	dbmap.AddTableWithName(Comment{}, "comments").SetKeys(true, "id")
	err = dbmap.CreateTablesIfNotExists()
	if err != nil {
		return nil, err
	}
	return dbmap, nil
}

func setupEcho() *echo.Echo {
	e := echo.New()
	e.Debug = true
	e.Logger.SetOutput(os.Stderr)

	return e
}

type Controller struct {
	dbmap *gorp.DbMap
}

func (controller *Controller) GetComment(c echo.Context) error {
	var comment Comment

	err := controller.dbmap.SelectOne(&comment, "SELECT * FROM comments where id = $1", c.Param("id"))
	if err != nil {
		if err != sql.ErrNoRows {
			c.Logger().Error("SelectOne: ", err)
			return c.String(http.StatusBadRequest, "SelectOne: "+err.Error())
		}
		return c.String(http.StatusNotFound, "Not Found")
	}
	return c.JSON(http.StatusOK, comment)
}

func (controller *Controller) ListComments(c echo.Context) error {
	var comments []Comment

	_, err := controller.dbmap.Select(&comments, "SELECT * FROM comments ORDER BY created desc LIMIT 10")
	if err != nil {
		c.Logger().Error("Select: ", err)
		return c.String(http.StatusBadRequest, "Select: "+err.Error())
	}
	return c.JSON(http.StatusOK, comments)
}

func (controller *Controller) InsertComments(c echo.Context) error {
	var comment Comment

	if err := c.Bind(&comment); err != nil {
		c.Logger().Error("Bind: ", err)
		return c.String(http.StatusBadRequest, "Bind: "+err.Error())
	}

	if err := controller.dbmap.Insert(&comment); err != nil {
		c.Logger().Error("Insert: ", err)
		return c.JSON(http.StatusBadRequest, "Insert: "+err.Error())
	}
	c.Logger().Infof("inserted comment: %v", comment.Id)
	return c.NoContent(http.StatusCreated)
}

func main() {
	dbmap, err := setupDB()
	if err != nil {
		log.Fatal(err)
	}
	controller := &Controller{dbmap: dbmap}

	e := setupEcho()

	e.GET("/api/comments/:id", controller.GetComment)
	e.GET("/api/comments", controller.ListComments)
	e.POST("/api/comments", controller.InsertComments)
	e.Logger.Fatal(e.Start(":8989"))
}
