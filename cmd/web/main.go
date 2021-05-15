package main

import (
	"crypto/tls"
	"database/sql"
	"flag"
	"github.com/Tike-Myson/snippetbox/pkg/models"
	"github.com/Tike-Myson/snippetbox/pkg/models/mysql"
	"github.com/golangcollege/sessions"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var (
	Reset = "\033[0m"
	Red   = "\033[31m"
	Green = "\033[32m"
)

type contextKey string

const contextKeyIsAuthenticated = contextKey("isAuthenticated")

type application struct {
	errorLog *log.Logger
	infoLog  *log.Logger
	session       *sessions.Session
	snippets interface {
		Insert(string, string, string) (int, error)
		Get(int) (*models.Snippet, error)
		Latest() ([]*models.Snippet, error)
	}
	templateCache map[string]*template.Template
	users         interface {
		Insert(string, string, string) error
		Authenticate(string, string) (int, error)
		Get(int) (*models.User, error)
		ChangePassword(int, string, string) error // Add the new method.
	}
}

func main() {
	addr := flag.String("addr", ":4000", "HTTP network address")
	dsn := flag.String("dsn", "nurtilek:nm2000kz@tcp(<127.0.0.1>:<3306>)/snippetbox?parseTime=true", "MySQL data source name")
	secret := flag.String("secret", "s6Ndh+pPbnzHbS*+9Pk8qGWhTzbpa@ge", "Secret key")
	flag.Parse()

	infoLog := log.New(os.Stdout, Green+"INFO\t"+Reset, log.Ldate|log.Ltime)
	errorLog := log.New(os.Stderr, Red+"ERROR\t"+Reset, log.Ldate|log.Ltime|log.Lshortfile)

	db, err := openDB(*dsn)
	if err != nil {
		errorLog.Fatal(err)
	}

	db.SetMaxOpenConns(100)
	db.SetMaxIdleConns(5)

	// We also defer a call to db.Close(), so that the connection pool is closed
	// before the main() function exits.
	defer db.Close()

	// Initialize a new template cache...
	templateCache, err := newTemplateCache("./ui/html/")
	if err != nil {
		errorLog.Fatal(err)
	}

	session := sessions.New([]byte(*secret))
	session.Lifetime = 12 * time.Hour
	session.Secure = true

	// Initialize a new instance of application containing the dependencies.
	app := &application{
		errorLog: errorLog,
		infoLog:  infoLog,
		session:       session,
		snippets: &mysql.SnippetModel{DB: db},
		templateCache: templateCache,
		users:         &mysql.UserModel{DB: db},
	}

	tlsConfig := &tls.Config{
		PreferServerCipherSuites: true,
		CurvePreferences:         []tls.CurveID{tls.X25519, tls.CurveP256},
	}

	srv := &http.Server{
		Addr:     *addr,
		ErrorLog: errorLog,
		Handler:  app.routes(),
		TLSConfig: tlsConfig,
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	infoLog.Printf("Server run on http://127.0.0.1%s\n", *addr)
	err = srv.ListenAndServe()
	//err = srv.ListenAndServeTLS("./tls/cert.pem", "./tls/key.pem")
	errorLog.Fatal(err)
}

// The openDB() function wraps sql.Open() and returns a sql.DB connection pool
// for a given DSN.
func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}