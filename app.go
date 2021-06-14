package main

import (
	"errors"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/timeout"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Config struct {
	Port       uint16     `yaml:"port"`
	Services   []Services `yaml:"services"`
	LogsKey    string     `yaml:"logsKey"`
	LogsSizeKb uint16     `yaml:"logsSizeKb"`
}

type Services struct {
	Name   string `yaml:"name"`
	Id     string `yaml:"id"`
	Script string `yaml:"script"`
}

var _Config Config
var busyMap sync.Map

func main() {
	logFilePath := initLogger()
	log.Infoln("App started...")
	log.Debugln("Logs dir:", logFilePath)
	initConfigDir()

	_Config = loadConfig()

	initHttpServer()
}

func execService(id string) (string, error) {
	busyFlag, _ := busyMap.Load(id)

	if busyFlag != nil {
		if busyFlag.(bool) {
			return "", errors.New("service is busy")
		}
	}
	busyMap.Store(id, true)

	for _, service := range _Config.Services {
		if service.Id != id {
			continue
		}
		log.Infoln(strings.Repeat("\n", 20))
		log.Infoln("Service \"" + service.Name + "\" started...")
		execFile, _ := filepath.Abs(path.Join(".", "config", service.Script))
		out, err := exec.Command("sh", execFile).Output()
		if err != nil {
			log.Errorln("Service", service.Name, "error:", err)
			return "", err
		}
		log.Infoln(string(out))
		log.Infoln("Service", service.Name, "finished...")
		busyMap.Store(id, false)
		return string(out), nil
	}
	busyMap.Store(id, false)
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

func readFile(fname string) string {
	file, err := os.Open(fname)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	var MaxLength = _Config.LogsSizeKb * 1000
	buf := make([]byte, MaxLength)
	stat, err := os.Stat(fname)
	if err != nil {
		return ""
	}
	start := stat.Size() - int64(MaxLength)
	_, err = file.ReadAt(buf, start)

	return string(buf)
}

func initHttpServer() {
	var HttpPort uint16
	if _Config.Port != 0 {
		HttpPort = _Config.Port
	} else {
		HttpPort = 9999
	}
	app := fiber.New(fiber.Config{
		//DisableStartupMessage: true,
	})

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello👋!\nUse /run/[service id]\nor /logs")
	})

	app.Get("/logs/"+_Config.LogsKey, func(c *fiber.Ctx) error {
		logsFile, _ := filepath.Abs(path.Join(".", "logs", "latest.log"))
		result := strings.TrimSpace(readFile(logsFile))
		return c.SendString(result)
	})

	app.Get("/run/:id", timeout.New(func(c *fiber.Ctx) error {
		res, err := execService(c.Params("id"))
		if err != nil {
			return c.SendString("Error: " + err.Error())
		}
		return c.SendString(res)
	}, 1*time.Hour))

	_ = app.Listen(":" + strconv.Itoa(int(HttpPort)))
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
	log.SetFormatter(&log.TextFormatter{
		DisableColors: true,
		DisableQuote:  true,
		FullTimestamp: true,
	})
	log.SetLevel(log.TraceLevel)

	return logsFile
}
