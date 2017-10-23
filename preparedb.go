package main

import (
	"flag"
	"os"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/term"
	"github.com/newtonsystems/agent-mgmt/app/models"

	mgo "gopkg.in/mgo.v2"
)

var debug = flag.Bool("debug", false, "turn on mongo debug")

func envString(env, fallback string) string {
	e := os.Getenv(env)
	if e == "" {
		return fallback
	}
	return e
}

func main() {
	// Color by level value
	colorFn := func(keyvals ...interface{}) term.FgBgColor {
		for i := 0; i < len(keyvals)-1; i += 2 {
			if keyvals[i] != "level" {
				continue
			}
			switch keyvals[i+1] {
			case "debug":
				return term.FgBgColor{Fg: term.DarkGray}
			case "info":
				return term.FgBgColor{Fg: term.DarkGreen}
			case "warn":
				return term.FgBgColor{Fg: term.Yellow, Bg: term.White}
			case "error":
				return term.FgBgColor{Fg: term.Red}
			case "crit":
				return term.FgBgColor{Fg: term.Gray, Bg: term.DarkRed}
			default:
				return term.FgBgColor{}
			}
		}
		return term.FgBgColor{}
	}

	// Logging domain.
	var logger log.Logger
	{
		//logger = log.NewLogfmtLogger(os.Stdout)
		logger = term.NewLogger(os.Stdout, log.NewLogfmtLogger, colorFn)
		logger = log.With(logger, "ts", log.DefaultTimestampUTC)
		logger = log.With(logger, "caller", log.DefaultCaller)
		logger = log.With(logger, "service", "agent-mgmt")
	}

	var mongoDB = envString("MONGO_DB", "test")

	mongoHosts := []string{
		"localhost:27017",
	}

	// We need this object to establish a session to our MongoDB.
	mongoDBDialInfo := &mgo.DialInfo{
		Addrs:    mongoHosts,
		Timeout:  60 * time.Second,
		Database: mongoDB,
		// TODO: Add auth to mongo
		//Username: MongoUsername,
		//Password: MongoPassword,
	}
	// Initialise mongodb connection
	// Create a session which maintains a pool of socket connections to our MongoDB.
	mongoSession, mongoLogger := models.NewMongoSession(mongoDBDialInfo, logger, *debug)
	defer mongoSession.Close()

	models.PrepareDB(mongoSession, mongoDB, mongoLogger)

}
