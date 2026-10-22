package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
)

const (
	listenAddr = ":9999"
	booksDir   = "/mnt/nas/Books"
	youtubeDir = "/mnt/nas/YT_video"
)

var (
	fileOpener   = getEnv("FILE_OPENER", "foliate")        // для книг
	folderOpener = getEnv("FOLDER_OPENER", "cosmic-files") // для папок
	videoOpener  = getEnv("VIDEO_OPENER", "vlc")           // для видео
)

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	app := fiber.New()

	app.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "OPTIONS"},
	}))

	app.Get("/health", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":        "ok",
			"service":       "books-opener",
			"file_opener":   fileOpener,
			"folder_opener": folderOpener,
		})
	})

	// Открыть файл
	app.Get("/open", func(c fiber.Ctx) error {
		name := c.Query("name")
		if name == "" {
			return c.Status(400).JSON(fiber.Map{"error": "name is required"})
		}
		if strings.Contains(name, "..") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
			return c.Status(400).JSON(fiber.Map{"error": "invalid name"})
		}

		filePath := filepath.Join(booksDir, name)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			return c.Status(404).JSON(fiber.Map{"error": "file not found", "path": filePath})
		}

		// Пробуем fileOpener, если не сработает — xdg-open
		if err := run(fileOpener, filePath); err != nil {
			log.Printf("⚠️ %s не сработал: %v, пробую xdg-open", fileOpener, err)
			if err := run("xdg-open", filePath); err != nil {
				return c.Status(500).JSON(fiber.Map{"error": err.Error()})
			}
		}

		log.Printf("📂 Открываю: %s", filePath)
		return c.JSON(fiber.Map{"status": "opened", "path": filePath})
	})

	// Открыть папку
	app.Get("/open-folder", func(c fiber.Ctx) error {
		if err := run(folderOpener, booksDir); err != nil {
			log.Printf("⚠️ %s не сработал: %v, пробую xdg-open", folderOpener, err)
			if err := run("xdg-open", booksDir); err != nil {
				return c.Status(500).JSON(fiber.Map{"error": err.Error()})
			}
		}

		log.Printf("📂 Открываю папку: %s", booksDir)
		return c.JSON(fiber.Map{"status": "opened", "path": booksDir})
	})

	// Открыть видео в плеере
	app.Get("/open-video", func(c fiber.Ctx) error {
		name := c.Query("name")
		if name == "" {
			return c.Status(400).JSON(fiber.Map{"error": "name is required"})
		}
		if strings.Contains(name, "..") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
			return c.Status(400).JSON(fiber.Map{"error": "invalid name"})
		}
		filePath := filepath.Join(youtubeDir, name)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			return c.Status(404).JSON(fiber.Map{"error": "file not found", "path": filePath})
		}
		if err := run(videoOpener, filePath); err != nil {
			log.Printf("⚠️ %s не сработал: %v, пробую xdg-open", videoOpener, err)
			if err := run("xdg-open", filePath); err != nil {
				return c.Status(500).JSON(fiber.Map{"error": err.Error()})
			}
		}
		log.Printf("▶️ Открываю видео: %s", filePath)
		return c.JSON(fiber.Map{"status": "opened", "path": filePath})
	})

	// Открыть медиа по полному пути (фильмы/сериалы)
	app.Get("/open-media", func(c fiber.Ctx) error {
		rawPath := c.Query("path")
		if rawPath == "" {
			return c.Status(400).JSON(fiber.Map{"error": "path is required"})
		}

		// Очищаем и делаем абсолютным до проверки allowlist
		path, err := filepath.Abs(filepath.Clean(rawPath))
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid path", "path": rawPath})
		}

		allowedPrefixes := []string{
			"/mnt/nas/root/FuzzyHD/Movies/",
			"/mnt/nas/root/FuzzyHD/TV/",
			"/mnt/nas/YT_video/",
		}
		ok := false
		for _, p := range allowedPrefixes {
			if strings.HasPrefix(path, p) {
				ok = true
				break
			}
		}
		if !ok {
			return c.Status(403).JSON(fiber.Map{"error": "path not allowed"})
		}

		// Разрешаем симлинки и проверяем снова, чтобы нельзя было выйти из разрешённых директорий
		if resolved, err := filepath.EvalSymlinks(path); err == nil {
			resolvedOk := false
			for _, p := range allowedPrefixes {
				if strings.HasPrefix(resolved, p) {
					resolvedOk = true
					break
				}
			}
			if !resolvedOk {
				return c.Status(403).JSON(fiber.Map{"error": "path not allowed", "resolved": resolved})
			}
			path = resolved
		}

		if _, err := os.Stat(path); err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "file not found", "path": path})
		}
		if err := run(videoOpener, path); err != nil {
			log.Printf("⚠️ %s не сработал: %v, пробую xdg-open", videoOpener, err)
			if err := run("xdg-open", path); err != nil {
				return c.Status(500).JSON(fiber.Map{"error": err.Error()})
			}
		}
		log.Printf("▶️ Открываю медиа: %s", path)
		return c.JSON(fiber.Map{"status": "opened", "path": path})
	})

	log.Printf("📂 books-opener на %s", listenAddr)
	log.Printf("   Папка с книгами: %s", booksDir)
	log.Printf("   Открываю файлы: %s", fileOpener)
	log.Printf("   Открываю папки: %s", folderOpener)
	log.Fatal(app.Listen(listenAddr))
}

func run(command string, args ...string) error {
	cmd := exec.Command(command, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	// Даём процессу немного времени: если он сразу завершился с ошибкой —
	// возвращаем ошибку, чтобы endpoint не отвечал "opened" при провале запуска.
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			return err
		}
		return nil
	case <-time.After(1500 * time.Millisecond):
		return nil // процесс ещё жив — считаем успешным запуском
	}
}
