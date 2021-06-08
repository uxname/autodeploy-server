package main

import (
	"errors"
	"fmt"
	"github.com/gofiber/fiber/v2"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
)

type Config struct {
	Services []Services `yaml:"services"`
}

type Services struct {
	Name   string `yaml:"name"`
	Id     string `yaml:"id"`
	Script string `yaml:"script"`
	Busy   bool
}

var _Config Config

func main() {
	logFilePath := initLogger()
	log.Infoln("App started...")
	log.Debugln("Logs dir:", logFilePath)
	initConfigDir()

	_Config = loadConfig()

	initHttpServer()
}

func execService(id string) (string, error) {
	for _, service := range _Config.Services {
		if service.Id != id {
			continue
		}
		if service.Busy {
			return "", errors.New("service is busy")
		}
		service.Busy = true
		log.Infoln("Service", service.Name, "started...")
		out, err := exec.Command("sh", service.Script).Output()
		if err != nil {
			log.Errorln("Service", service.Name, "error:", err)
			service.Busy = false
			return "", err
		}
		log.Infoln(string(out))
		log.Infoln("Service", service.Name, "finished...")
		service.Busy = false
		return string(out), nil
	}
	return "", errors.New("service not found")
}

func loadConfig() Config {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(path.Join(".", "config"))
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

	var config Config
	err = viper.Unmarshal(&config)
	if err != nil {
		panic(err)
	}
	return config
}

func initHttpServer() {
	const HttpPort = 9999
	app := fiber.New(fiber.Config{
		//DisableStartupMessage: true,
	})

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello👋!\nUse /run/[service id]")
	})

	app.Get("/run/:id", func(c *fiber.Ctx) error {
		res, err := execService(c.Params("id"))
		if err != nil {
			return c.SendString("Error: " + err.Error())
		}
		return c.SendString(res)
	})

	_ = app.Listen(":" + strconv.Itoa(HttpPort))
}

func initConfigDir() {
	configDir, err := filepath.Abs(filepath.Join(".", "config"))
	err = os.MkdirAll(configDir, os.ModePerm)
	if err != nil {
		panic("Can't create config dir")
	}
	log.Debugln("Config dir:", configDir)
}

func initLogger() string {
	logsFile, _ := filepath.Abs(path.Join(".", "logs", "latest.log"))
	fileLogger := &lumberjack.Logger{
		Filename:   logsFile,
		MaxSize:    10, // megabytes
		MaxBackups: 5,
		Compress:   true, // disabled by default
	}

	defer func(fileLogger *lumberjack.Logger) {
		_ = fileLogger.Close()
	}(fileLogger)

	mw := io.MultiWriter(os.Stdout, fileLogger)
	log.SetOutput(mw)
	log.SetFormatter(&log.TextFormatter{ForceColors: true, FullTimestamp: true})
	log.SetLevel(log.TraceLevel)

	return logsFile
}
