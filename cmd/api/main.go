package main

import (
	"context"
	"net/http"

	"geocoding-api/internal/config"
	"geocoding-api/internal/handler"
	"geocoding-api/internal/repository"
	"geocoding-api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

func main() {
	config, err := config.LoadConfig("./configs")
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

	r.Run(config.ServerAddress)
}
