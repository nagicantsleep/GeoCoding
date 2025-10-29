package main

import (
	"context"
	"net/http"
	"path/filepath"

	"geocoding-api/internal/config"
	"geocoding-api/internal/handler"
	"geocoding-api/internal/repository"
	"geocoding-api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	ginSwagger "github.com/swaggo/gin-swagger"
	files "github.com/swaggo/files"
)

// @title Geocoding API
// @version 1.0
// @description A geocoding service API for Japanese addresses
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /

func main() {
	config, err := config.LoadConfig(filepath.Join(".", "configs"))
	if err != nil {
		log.Fatal().Err(err).Msg("cannot load config")
	}

	// Database connection
	conn, err := pgxpool.New(context.Background(), config.DBSource)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot connect to db")
	}
	defer conn.Close()

	// Initialize layers
	repo := repository.NewRepository(conn)

	geoCodeService := service.NewGeoCodeService(repo)
	reverseGeocodeService := service.NewReverseGeoCodeService(repo)

	geoCodeHandler := handler.NewGeoCodeHandler(geoCodeService)
	reverseGeocodeHandler := handler.NewReverseGeocodeHandler(reverseGeocodeService)

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	r.GET("/geocode", geoCodeHandler.GeoCode)
	r.GET("/reverse-geocode", reverseGeocodeHandler.ReverseGeocode)

	// Swagger UI route
	r.GET("/swagger/*any", ginSwagger.WrapHandler(files.Handler))

	r.Run(config.ServerAddress)
}
