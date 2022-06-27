package core

//GO-REST-FRAMEWORK__________________________________________________________________
//  |  |                                                                             \
//  |  |                                                                              |--- group helpers to one directory
//  |  \___ Core--\                                                                   |
//  |  |           |                                                                  |
//  |  |           |--- Logs system                                                   |
//  |  |           |                                                                  |
//  |  |           |--- New config                                                    |
//  |  |           |                                                                  |
//  |  |           |--- i18n                                                          |
//  |  |
//  |  \___ User--\
//  |  |          |
//  |  |          |--- Oauth2
//  |  |          |
//  |  |          |--- User Keyword System (universal type/subtype/role/group tool)
//  |  |          |      |
//  |  |          |      |--- admin add list of keywords
//  |  |          |               creat
//  |  |          |                   by admin in admin panel
//  |  |          |               atach
//  |  |          |                   by admin
//  |  |          |                   by option from registration
//  |  |          |                   by special function
//  |  |          |               check
//  |  |          |                   by user method
//  |  |          |                   by endpoint for api
//  |  |          |
//  |  |          |--- Profile
//  |  |          |
//  |  |
//  |  \___ System msgs
//  |
//  |
//  \___ Content Element

import (
	"crypto/sha256"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/go-rest-framework/arrayHelper"
	"github.com/go-rest-framework/mailHelper"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	_ "github.com/jinzhu/gorm/dialects/postgres"
)

var TokenSigningKey []byte

type Config struct {
	Dbtype          string
	Dbhost          string
	Dbname          string
	Dbuser          string
	Dbpass          string
	TokenSigningKey string
	MailLogin       string
	MailPassword    string
	MailServer      string
	MailPort        string
	UploadsPath     string
	WebRootPath     string
}

type App struct {
	DB     *gorm.DB
	R      *mux.Router
	Mail   mailHelper.Mailer
	Config Config
	IsTest bool
}

func (a *App) Init() {
	var db *gorm.DB
	var err error
	if len(os.Args) > 1 {
		if os.Args[1] == "test" {
			a.IsTest = true
			fmt.Printf("%s\n", "!!! Service run in TEST MODE !!!")
		}
	}

	TokenSigningKey = []byte(a.Config.TokenSigningKey)
	if a.Config.Dbtype == "mysql" {
		connectstr := fmt.Sprintf(
			"%s:%s@(%s)/%s?charset=utf8&parseTime=True&loc=Local",
			a.Config.Dbuser,
			a.Config.Dbpass,
			a.Config.Dbhost,
			a.Config.Dbname,
		)
		db, err = gorm.Open("mysql", connectstr)
		if err != nil {
			panic(err)
		}
	} else if a.Config.Dbtype == "postgres" {
		connectstr := fmt.Sprintf(
			"host=%s user=%s dbname=%s password=%s sslmode=disable",
			a.Config.Dbhost,
			a.Config.Dbuser,
			a.Config.Dbname,
			a.Config.Dbpass,
		)

		db, err = gorm.Open("postgres", connectstr)
		if err != nil {
			panic(err)
		}
	}
	a.DB = db
	a.R = mux.NewRouter().StrictSlash(false)
	a.Mail.Email = a.Config.MailLogin
	a.Mail.Pass = a.Config.MailPassword
	a.Mail.Server = a.Config.MailServer
	a.Mail.Port = a.Config.MailPort
}

func (a *App) Protect(next func(w http.ResponseWriter, r *http.Request), roles []string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.RequestURI)
		tokenString := r.Header.Get("Authorization")
		if len(tokenString) == 0 {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Missing Authorization Header"))
			return
		}
		tokenString = strings.Replace(tokenString, "Bearer ", "", 1)
		claims, err := a.CheckToken(tokenString)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Error verifying JWT token: " + err.Error()))
			return
		}
		id := claims.(jwt.MapClaims)["id"].(string)
		name := claims.(jwt.MapClaims)["name"].(string)
		role := claims.(jwt.MapClaims)["role"].(string)
		status := claims.(jwt.MapClaims)["status"].(string)

		if !arrayHelper.Include(roles, role) {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("You are not allowed to perform this action.(" + role + ")"))
			return
		}

		if status != "active" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Only active users can perform this action"))
			return
		}

		r.Header.Set("id", id)
		r.Header.Set("name", name)
		r.Header.Set("role", role)

		http.HandlerFunc(next).ServeHTTP(w, r)
		log.Println("close")
	}
}

func (a *App) GenToken(id, login, role *string, status *string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":     id,
		"name":   login,
		"role":   role,
		"status": status,
	})
	tokenString, err := token.SignedString(TokenSigningKey)
	return tokenString, err
}

func (a *App) CheckToken(tokenString string) (jwt.Claims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		return TokenSigningKey, nil
	})
	if err != nil {
		return nil, err
	}
	return token.Claims, err
}

func (a *App) Run(addrs string) {
	server := &http.Server{
		Addr:    addrs,
		Handler: a.R,
	}

	server.ListenAndServe()

	defer a.DB.Close()
}

func (a *App) ToSum256(s string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(s)))
}
