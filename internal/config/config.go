package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Kafka struct {
	Brokers string
	Topic   string
}

type Postgres struct {
	ConnectionString string
}

type JWT struct {
	Secret string
}

type Config struct {
	Kafka    Kafka
	Postgres Postgres
	HTTPAddr string
	JWT      JWT
}

func GetConfig() (Config, error) {
	err := godotenv.Load()

	// Db
	dbHost := os.Getenv("DB_HOST")
	dbName := os.Getenv("DB_DATABASE")
	dbUsername := os.Getenv("DB_USERNAME")
	dbPassword := os.Getenv("DB_PASSWORD")
	connectionString := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", dbUsername, dbPassword, dbHost, dbName)

	// Kafka
	brokers := os.Getenv("KAFKA_BROKERS")
	topic := os.Getenv("KAFKA_TOPIC")

	httpAddr := os.Getenv("HTTP_ADDRESS")

	jwtSecret := os.Getenv("JWT_SECRET")
	config := Config{
		Postgres: Postgres{ConnectionString: connectionString},
		Kafka:    Kafka{Brokers: brokers, Topic: topic},
		HTTPAddr: httpAddr,
		JWT:      JWT{Secret: jwtSecret},
	}

	return config, err
}
