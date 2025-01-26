package api

import (
	"context"
	"fmt"
	"gateway/configuration"
	"gateway/db"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	dockertest "github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	postgresClient      *gorm.DB
	postgresPool        *dockertest.Pool
	postgresResource    *dockertest.Resource
	once                sync.Once
	CollectionsToCreate = []string{"ingredient", "price", "shop"}
	DBName              = "gateway"
	DBUser              = "root"
	DBPassword          = "password"
	DBPort              = "?"
	DBHost              = "localhost"
	// DBUri               = fmt.Sprintf("postgresdb://%s:%s@%s:%s/%s", DBUser, DBPassword, DBHost, DBPort, DBName)
)

func SeedDatabase(pg *gorm.DB) {
	err := db.AutoMigrate(pg)
	if err != nil {
		log.Fatalf("Failed to migrate db: %v", err)
	}
}

// InitTestPostgres initializes a single PostgresDB instance for all tests
func InitTestPostgres() (*gorm.DB, error) {
	var initErr error
	once.Do(func() {
		// Create a new pool
		pool, err := dockertest.NewPool("")
		if err != nil {
			initErr = fmt.Errorf("could not construct pool: %w", err)
			return
		}

		postgresPool = pool

		// Set a timeout for docker operations
		pool.MaxWait = time.Second * 30

		// Start PostgresDB container
		resource, err := pool.RunWithOptions(&dockertest.RunOptions{
			Repository: "bitnami/postgresql",
			Tag:        "latest",
			Env: []string{
				fmt.Sprintf("POSTGRESQL_DATABASE=%v", DBName),
				fmt.Sprintf("POSTGRESQL_USERNAME=%v", DBUser),
				fmt.Sprintf("POSTGRESQL_PASSWORD=%v", DBPassword),
			},
		}, func(config *docker.HostConfig) {
			config.AutoRemove = true
			config.RestartPolicy = docker.RestartPolicy{Name: "no"}
		})

		if err != nil {
			initErr = fmt.Errorf("could not start resource: %w", err)
			return
		}

		postgresResource = resource
		DBPort = resource.GetPort("5432/tcp")
		dsn := fmt.Sprintf("host=%v port=%v user=%v password=%v dbname=%v sslmode=%v TimeZone=%v ",
			DBHost,
			DBPort,
			DBUser,
			DBPassword,
			DBName,
			"disable",
			"Europe/Paris")

		gormLogger := db.NewGormLogger()

		// Initialize postgres client
		logger.Info("Connecting to DB: " + dsn)
		// Retry connection with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
		defer cancel()

		ticker := time.NewTicker(300 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				initErr = fmt.Errorf("timeout waiting for postgresdb to be ready")
				return
			case <-ticker.C:
				client, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
					Logger: gormLogger,
				})
				if err != nil {
					continue
				}

				// Try to ping
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()
				db, err := client.WithContext(ctx).DB()
				if err != nil {
					continue
				}

				if err := db.Ping(); err != nil {
					_ = db.Close()
					continue
				}

				postgresClient = client
				return
			}
		}
	})

	return postgresClient, initErr
}

// CleanupDatabase removes all data from the test database
func CleanupDatabase(t *testing.T, client *gorm.DB) {
	// client.Exec("DROP TABLE users")
	// client.Exec("DROP TABLE tokens")
}

func setupTest(t *testing.T) (*ApiHandler, func()) {
	t.Helper()

	// Use existing PostgresDB instance
	client := postgresClient
	if client == nil {
		t.Fatal("PostgresDB client not initialized")
	}

	// Clean the database
	//CleanupDatabase(t, client)

	// Initialize the database
	//SeedDatabase(client)

	// Create API handler
	conf := &configuration.Configuration{
		ListenAddress: "localhost",
		ListenPort:    "3000",
		LogLevel:      logrus.DebugLevel,
		DBName:        DBName,
		DBUser:        DBUser,
		DBPassword:    DBPassword,
		DBPort:        DBPort,
		DBHost:        DBHost,
		DBSSLMode:     "disable",
		DBTimezone:    "Europe/Paris",
	}

	db, err := db.NewPostgresHandler(conf)
	if err != nil {
		t.Fatalf("Failed to create PostgresDB handler: %v", err)
	}
	api := NewApiHandler(db, nil, conf)

	// Return cleanup function
	return api, func() {
		CleanupDatabase(t, client)
	}
}

func TestMain(m *testing.M) {
	// Setup
	client, err := InitTestPostgres()
	SeedDatabase(client)
	if err != nil {
		log.Fatalf("Could not start PostgresDB: %s", err)
	}

	// Run tests
	code := m.Run()

	// Cleanup
	if client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		pg, err := client.WithContext(ctx).DB() // Disconnect(ctx)
		if err != nil {
			log.Fatalf("Could not disconnect from PostgresDB: %s", err)
		}
		pg.Close()
	}
	if postgresPool != nil && postgresResource != nil {
		_ = postgresPool.Purge(postgresResource)
	}

	os.Exit(code)
}

func TestDB(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		test func(t *testing.T)
	}{

		{
			name: "Create and retrieve a user in the DB",
			test: func(t *testing.T) {
				api, cleanup := setupTest(t)
				defer cleanup()
				user1 := db.UserRequest{
					Email:    "user1@test.me",
					Username: "user1",
					Password: "password",
				}
				u1, err := api.dbh.CreateUser(&user1)
				if err != nil {
					t.Fatalf("Failed to insert user: %v", err)
				}

				// Check if the user has an id, username, password and email
				if u1.GetUsername() != "user1" || u1.GetPassword() != "password" || u1.GetEmail() != "user1@test.me" {
					t.Fatalf("User not inserted: %v", u1)
				}

				// Retrieve the user from the DB
				u2, err := api.dbh.GetUsername("user1")
				if err != nil {
					t.Fatalf("Failed to get user: %v", err)
				}
				if u2.GetUsername() != "user1" || u2.GetPassword() != "password" || u2.GetEmail() != "user1@test.me" {
					t.Fatalf("User not retrieved: %v", u2)
				}
			},
		},
		{
			name: "Create and retrieve a token in the DB",
			test: func(t *testing.T) {
				api, cleanup := setupTest(t)
				defer cleanup()
				userId := "900"
				value := "token1"
				token1 := db.TokenRequest{
					UserID:         userId,
					Value:          value,
					ExpirationDate: time.Now().UTC().Add(time.Hour),
				}

				t1, err := api.dbh.UpsertToken(&token1)
				if err != nil {
					t.Fatalf("Failed to insert token: %v", err)
				}

				// Check if the token has an id, value, expiration date and user id
				t1expRound := t1.GetExpirationDate().Round(time.Second)
				token1expRound := token1.ExpirationDate.Round(time.Second)
				if t1.GetValue() != value || t1expRound != token1expRound || t1.GetUserID() != userId {
					t.Fatalf("Token not inserted: %v", t1)
				}

				t2, err := api.dbh.GetTokenUser(value, userId)
				if err != nil {
					t.Fatalf("Failed to get token: %v", err)
				}
				if t2.GetValue() != value || t2.GetUserID() != userId {
					t.Fatalf("Token not retrieved: %v", t2)
				}

				// Check that the expiration date matches rounded to the millisecond
				t2Round := t2.GetExpirationDate().Round(time.Millisecond)
				token1Round := t1.GetExpirationDate().Round(time.Millisecond)

				tolerance := time.Millisecond // Adjust tolerance as needed
				if t2Round.Sub(token1Round).Abs() > tolerance {
					t.Fatalf("Token expiration date doesn't match t1: %v, t2: %v", t2Round, token1Round)
				}
			},
		},
		{
			name: "Upsert Token change the good token and not the other",
			test: func(t *testing.T) {
				api, cleanup := setupTest(t)
				defer cleanup()
				userId1 := "1004"
				value1 := "token1"
				exp1 := time.Now().UTC().Add(time.Hour)
				userId2 := "1005"
				value2 := "token2"
				exp2 := time.Now().UTC().Add(time.Hour * 2)

				t1 := &db.TokenRequest{
					UserID:         userId1,
					Value:          value1,
					ExpirationDate: exp1,
				}
				t2 := &db.TokenRequest{
					UserID:         userId2,
					Value:          value2,
					ExpirationDate: exp2,
				}

				_, err := api.dbh.UpsertToken(t1)
				if err != nil {
					t.Fatalf("Failed to insert token: %v", err)
				}
				_, err = api.dbh.UpsertToken(t2)
				if err != nil {
					t.Fatalf("Failed to insert token: %v", err)
				}

				// Change the value of the first token and upsert it
				t1.Value = "newToken"
				newExp := time.Now().UTC().Add(time.Hour * 3)
				t1.ExpirationDate = newExp
				_, err = api.dbh.UpsertToken(t1)
				if err != nil {
					t.Fatalf("Failed to insert token: %v", err)
				}
				// Ensure the first token has changed
				t5, err := api.dbh.GetTokenUser("newToken", userId1)
				if err != nil {
					t.Fatalf("Failed to get token: %v", err)
				}
				if t5.GetValue() != "newToken" || t5.GetUserID() != userId1 {
					t.Fatalf("Token has changed. Expected value=%v, userId=%v, got %v", "newToken", userId1, t1)
				}

				// Retrieve the 2nd token and check if it's the same
				t6, err := api.dbh.GetTokenUser(value2, userId2)
				if err != nil {
					t.Fatalf("Failed to get token: %v", err)
				}
				if t6.GetValue() != value2 || t6.GetUserID() != userId2 {
					t.Fatalf("Token has changed expected value=%v, userId=%v, got %v", value2, userId2, t2)
				}
			},
		},
		{
			name: "Delete a token in the DB",
			test: func(t *testing.T) {
				api, cleanup := setupTest(t)
				defer cleanup()
				userId := "11"
				value := "token"
				token := db.TokenRequest{
					UserID:         userId,
					Value:          value,
					ExpirationDate: time.Now().UTC().Add(time.Hour),
				}
				_, err := api.dbh.UpsertToken(&token)
				if err != nil {
					t.Fatalf("Failed to insert token: %v", err)
				}
				err = api.dbh.DeleteToken(userId)
				if err != nil {
					t.Fatalf("Failed to delete token: %v", err)
				}

				// Check if the token has been deleted
				t2, err := api.dbh.GetTokenUser(value, userId)
				if err == nil {
					t.Fatalf("Token not deleted: %v", t2)
				}
			},
		},
	}
	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.test(t)
		})
	}
}
