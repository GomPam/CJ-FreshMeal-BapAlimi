package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	githubOwner = "GomPam"
	githubRepo  = "CJ-FreshMeal-BapAlimi"
	githubAPI   = "https://api.github.com/repos/" + githubOwner + "/" + githubRepo + "/releases/latest"
	updateTmp   = "update_tmp.exe"
	oldSuffix   = ".old"
)

type UpdateInfo struct {
	Available   bool   `json:"available"`
	CurrentVer  string `json:"currentVer"`
	LatestVer   string `json:"latestVer"`
	ReleaseURL  string `json:"releaseUrl"`
	DownloadURL string `json:"downloadUrl"`
	ReleaseNote string `json:"releaseNote"`
}

type UpdateProgress struct {
	Phase   string `json:"phase"`
	Percent int    `json:"percent"`
	Message string `json:"message"`
}

type ghRelease struct {
	TagName    string    `json:"tag_name"`
	HTMLURL    string    `json:"html_url"`
	Body       string    `json:"body"`
	Assets     []ghAsset `json:"assets"`
}

type ghAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func compareVersions(current, latest string) bool {
	current = strings.TrimPrefix(current, "v")
	latest = strings.TrimPrefix(latest, "v")
	cParts := strings.Split(current, ".")
	lParts := strings.Split(latest, ".")
	maxLen := len(cParts)
	if len(lParts) > maxLen {
		maxLen = len(lParts)
	}
	for i := 0; i < maxLen; i++ {
		var c, l int
		if i < len(cParts) {
			c, _ = strconv.Atoi(cParts[i])
		}
		if i < len(lParts) {
			l, _ = strconv.Atoi(lParts[i])
		}
		if l > c {
			return true
		}
		if l < c {
			return false
		}
	}
	return false
}

func (a *App) CheckForUpdate() (*UpdateInfo, error) {
	info := &UpdateInfo{CurrentVer: appVersion}

	if appVersion == "dev" {
		return info, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubAPI, nil)
	if err != nil {
		return nil, fmt.Errorf("요청 생성 실패: %w", err)
	}
	req.Header.Set("User-Agent", "BapAlimi/"+appVersion)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("서버 연결 실패: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API 오류 (HTTP %d)", resp.StatusCode)
	}

	var release ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("응답 파싱 실패: %w", err)
	}

	latestVer := strings.TrimPrefix(release.TagName, "v")
	info.LatestVer = latestVer
	info.ReleaseURL = release.HTMLURL
	info.ReleaseNote = release.Body

	for _, asset := range release.Assets {
		if strings.HasSuffix(strings.ToLower(asset.Name), ".exe") {
			info.DownloadURL = asset.BrowserDownloadURL
			break
		}
	}

	info.Available = compareVersions(appVersion, latestVer) && info.DownloadURL != ""
	return info, nil
}

func (a *App) PerformUpdate(downloadURL string) error {
	if downloadURL == "" {
		return fmt.Errorf("다운로드 URL이 없습니다")
	}

	tmpPath := a.configPath(updateTmp)
	if err := a.downloadFile(downloadURL, tmpPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("다운로드 실패: %w", err)
	}

	if err := validateExe(tmpPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("파일 검증 실패: %w", err)
	}

	exePath, err := os.Executable()
	if err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("현재 실행 파일 경로 확인 실패: %w", err)
	}
	exePath, _ = filepath.EvalSymlinks(exePath)
	oldPath := exePath + oldSuffix

	os.Remove(oldPath)

	a.emitProgress("installing", 0, "파일 교체 중...")
	if err := os.Rename(exePath, oldPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("기존 파일 이름 변경 실패: %w", err)
	}

	if err := os.Rename(tmpPath, exePath); err != nil {
		os.Rename(oldPath, exePath)
		return fmt.Errorf("새 파일 배치 실패: %w", err)
	}

	a.emitProgress("restarting", 100, "재시작 중...")

	cmd := exec.Command(exePath)
	cmd.Dir = filepath.Dir(exePath)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("재시작 실패: %w", err)
	}

	go func() {
		time.Sleep(500 * time.Millisecond)
		a.QuitApp()
	}()

	return nil
}

func (a *App) OpenReleasePage(url string) {
	wailsRuntime.BrowserOpenURL(a.ctx, url)
}

func (a *App) downloadFile(url, destPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "BapAlimi/"+appVersion)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()

	total := resp.ContentLength
	var downloaded int64
	buf := make([]byte, 32*1024)
	lastEmit := time.Time{}

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, wErr := f.Write(buf[:n]); wErr != nil {
				return wErr
			}
			downloaded += int64(n)
			if total > 0 && time.Since(lastEmit) > 200*time.Millisecond {
				pct := int(downloaded * 100 / total)
				a.emitProgress("downloading", pct, fmt.Sprintf("%d%%", pct))
				lastEmit = time.Now()
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}

	a.emitProgress("downloading", 100, "다운로드 완료")
	return nil
}

func validateExe(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	header := make([]byte, 2)
	if _, err := io.ReadFull(f, header); err != nil {
		return fmt.Errorf("헤더 읽기 실패: %w", err)
	}
	if header[0] != 'M' || header[1] != 'Z' {
		return fmt.Errorf("유효하지 않은 실행 파일")
	}

	info, err := f.Stat()
	if err != nil {
		return err
	}
	if info.Size() < 1*1024*1024 {
		return fmt.Errorf("파일 크기가 비정상적으로 작습니다 (%d bytes)", info.Size())
	}

	return nil
}

func (a *App) emitProgress(phase string, percent int, message string) {
	wailsRuntime.EventsEmit(a.ctx, "update:progress", UpdateProgress{
		Phase:   phase,
		Percent: percent,
		Message: message,
	})
}

func (a *App) cleanupOldBinary() {
	exePath, err := os.Executable()
	if err != nil {
		return
	}
	exePath, _ = filepath.EvalSymlinks(exePath)
	oldPath := exePath + oldSuffix
	if _, err := os.Stat(oldPath); err == nil {
		os.Remove(oldPath)
	}
}
