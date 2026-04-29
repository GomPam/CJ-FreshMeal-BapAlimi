package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/sys/windows/registry"

	msalcache "github.com/AzureAD/microsoft-authentication-library-for-go/apps/cache"
	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/public"
	"github.com/energye/systray"
)

const appTitle = "BapAlimi"

var appVersion = "dev"

const (
	msalClientID  = "14d82eec-204b-4c2f-b7e8-296a70dab67e"
	msalAuthority = "https://login.microsoftonline.com/organizations"
	graphBaseURL  = "https://graph.microsoft.com/v1.0"
	mealBaseURL   = "https://front.cjfreshmeal.co.kr/meal/v1"
	storeBaseURL  = "https://front.cjfreshmeal.co.kr/store/v1"
)

var msalScopes = []string{
	"Chat.ReadWrite",
}

// --- 운영 설정 (회사/운영 방침에 따라 조정) ---

var (
	// 기본 전송 시간 (최초 설치 또는 설정 초기화 시 적용)
	defaultSendTimes = []string{"11:20", "17:20"}

	// 식사 종료 시간 (시, 해당 시간 이후 UI 접힘 및 Adaptive Card 흐림 처리)
	mealEndHours = map[string]int{
		"1": 10, // 조식
		"2": 14, // 중식
		"3": 20, // 석식
	}

	// 썸네일 자동 갱신 시간대 (분, 자정 기준)
	// 해당 시간대에 오늘 식단을 보고 있으면 5분 간격 자동 새로고침
	thumbRefreshWindows = []map[string]int{
		{"start": 11*60 + 15, "end": 11*60 + 35}, // 중식 11:15~11:35
		{"start": 17*60 + 15, "end": 17*60 + 35}, // 석식 17:15~17:35
	}
)

type sentRecord struct {
	MessageID string `json:"messageId"`
	Date      string `json:"date"`
}

type LogEntry struct {
	Time    string `json:"time"`
	Level   string `json:"level"`
	Action  string `json:"action"`
	Target  string `json:"target"`
	Status  int    `json:"status"`
	Message string `json:"message"`
}

const maxLogs = 200

type App struct {
	ctx           context.Context
	msalApp       public.Client
	account       public.Account
	loggedIn      bool
	config        *AppConfig
	mu            sync.Mutex
	logMu         sync.Mutex
	stopCh        chan struct{}
	pendingDC     *public.DeviceCode
	windowVisible bool
	quitting      bool
	lastToggle    time.Time
	sentRecords   map[string]*sentRecord
	logs          []LogEntry
}

type ConfigTarget struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Type  string `json:"type"`
	Group string `json:"group"`
}

type AppConfig struct {
	StoreIdx     string         `json:"storeIdx"`
	StoreName    string         `json:"storeName"`
	Targets      []ConfigTarget `json:"targets"`
	Times        []string       `json:"times"`
	WindowX      int            `json:"windowX,omitempty"`
	WindowY      int            `json:"windowY,omitempty"`
	HasWindowPos bool           `json:"hasWindowPos,omitempty"`
}

type StoreSearchResult struct {
	TotalCount int         `json:"totalCount"`
	StoreList  []StoreItem `json:"storeList"`
}

type StoreItem struct {
	Idx        string `json:"idx"`
	Name       string `json:"name"`
	Address    string `json:"address"`
	WorkStatus string `json:"workStatus"`
	WorkTime   string `json:"workTime"`
	HldTxt     string `json:"hldTxt"`
}

type DeviceCode struct {
	UserCode        string `json:"UserCode"`
	VerificationURL string `json:"VerificationURL"`
}

type SendTarget struct {
	ID    string `json:"ID"`
	Name  string `json:"Name"`
	Type  string `json:"Type"`
	Group string `json:"Group"`
}

type Schedule struct {
	Targets []ConfigTarget `json:"targets"`
	Times   []string       `json:"times"`
}

func (a *App) GetDefaultTimes() []string {
	return defaultSendTimes
}

func (a *App) GetMealEndHours() map[string]int {
	return mealEndHours
}

func (a *App) GetThumbRefreshWindows() []map[string]int {
	return thumbRefreshWindows
}

func (a *App) GetAppVersion() string {
	return appVersion
}

func NewApp() *App {
	return &App{
		sentRecords: make(map[string]*sentRecord),
		logs:        make([]LogEntry, 0),
	}
}

func (a *App) addLog(level, action, target string, status int, message string) {
	entry := LogEntry{
		Time:    time.Now().Format("15:04:05"),
		Level:   level,
		Action:  action,
		Target:  target,
		Status:  status,
		Message: message,
	}
	a.logMu.Lock()
	a.logs = append(a.logs, entry)
	if len(a.logs) > maxLogs {
		a.logs = a.logs[len(a.logs)-maxLogs:]
	}
	a.logMu.Unlock()

	f, err := os.OpenFile(a.configPath("app.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		fmt.Fprintf(f, "%s [%s] %s target=%s status=%d %s\n",
			time.Now().Format("2006-01-02 15:04:05"), level, action, target, status, message)
		f.Close()
	}
}

func (a *App) GetLogs() []LogEntry {
	a.logMu.Lock()
	defer a.logMu.Unlock()
	copied := make([]LogEntry, len(a.logs))
	copy(copied, a.logs)
	return copied
}

func (a *App) ClearLogs() {
	a.logMu.Lock()
	a.logs = make([]LogEntry, 0)
	a.logMu.Unlock()
}

func (a *App) loadSentRecords() {
	data, err := os.ReadFile(a.configPath("sent_records.json"))
	if err != nil {
		return
	}
	var records map[string]*sentRecord
	if json.Unmarshal(data, &records) == nil && records != nil {
		today := time.Now().Format("2006-01-02")
		for k, v := range records {
			if v.Date == today {
				a.sentRecords[k] = v
			}
		}
		a.addLog("INFO", "INIT", "", 0, fmt.Sprintf("sentRecords 로드: %d건 (오늘)", len(a.sentRecords)))
	}
}

func (a *App) saveSentRecords() {
	data, err := json.MarshalIndent(a.sentRecords, "", "  ")
	if err != nil {
		return
	}
	os.WriteFile(a.configPath("sent_records.json"), data, 0644)
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.cleanupOldBinary()
	a.initMSAL()
	a.loadConfig()
	a.loadSentRecords()
	a.fixAutoStartPath()
	a.trySilentLogin()
	a.startScheduler()
	a.setupTray()
}

func (a *App) shutdown(_ context.Context) {
	if a.stopCh != nil {
		close(a.stopCh)
	}
	systray.Quit()
}

func (a *App) initMSAL() {
	cacheFile := a.configPath("token_cache.json")
	app, err := public.New(msalClientID,
		public.WithAuthority(msalAuthority),
		public.WithCache(&fileCache{file: cacheFile}),
	)
	if err != nil {
		return
	}
	a.msalApp = app
}

func (a *App) trySilentLogin() {
	accounts, err := a.msalApp.Accounts(context.Background())
	if err != nil || len(accounts) == 0 {
		return
	}
	a.account = accounts[0]
	result, err := a.msalApp.AcquireTokenSilent(context.Background(), msalScopes,
		public.WithSilentAccount(a.account),
	)
	if err != nil {
		return
	}
	if result.AccessToken != "" {
		a.loggedIn = true
	}
}

// --- Meal API ---

func (a *App) getStoreIdx() string {
	if a.config != nil && a.config.StoreIdx != "" {
		return a.config.StoreIdx
	}
	return ""
}

func (a *App) GetTodayMeal() (map[string]interface{}, error) {
	idx := a.getStoreIdx()
	if idx == "" {
		return nil, fmt.Errorf("식당이 설정되지 않았습니다")
	}
	url := fmt.Sprintf("%s/today-all-meal?storeIdx=%s", mealBaseURL, idx)
	return a.fetchMealAPI(url)
}

func (a *App) GetWeekMeal(weekType int) (map[string]interface{}, error) {
	idx := a.getStoreIdx()
	if idx == "" {
		return nil, fmt.Errorf("식당이 설정되지 않았습니다")
	}
	url := fmt.Sprintf("%s/week-meal?storeIdx=%s&weekType=%d", mealBaseURL, idx, weekType)
	return a.fetchMealAPI(url)
}

func (a *App) fetchMealAPI(url string) (map[string]interface{}, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	data, ok := result["data"].(map[string]interface{})
	if !ok {
		return map[string]interface{}{}, nil
	}
	return data, nil
}

// --- Store Search ---

func (a *App) SearchStore(keyword string, page int) (*StoreSearchResult, error) {
	if keyword == "" {
		return nil, fmt.Errorf("검색어를 입력해주세요")
	}
	if page < 1 {
		page = 1
	}
	reqURL := fmt.Sprintf("%s/search-store?page=%d&schKey=%s&isList=false", storeBaseURL, page, url.QueryEscape(keyword))
	resp, err := http.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("식당 검색 실패: %w", err)
	}
	defer resp.Body.Close()

	var raw struct {
		Status string `json:"status"`
		Data   struct {
			TotalCount int `json:"totalCount"`
			StoreList  []StoreItem `json:"storeList"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("응답 파싱 실패: %w", err)
	}
	return &StoreSearchResult{
		TotalCount: raw.Data.TotalCount,
		StoreList:  raw.Data.StoreList,
	}, nil
}

func (a *App) SetStore(idx, name string) error {
	a.mu.Lock()
	if a.config == nil {
		a.config = &AppConfig{}
	}
	a.config.StoreIdx = idx
	a.config.StoreName = name
	a.mu.Unlock()
	a.addLog("INFO", "STORE", idx, 0, "식당 설정: "+name)
	return a.saveConfig()
}

func (a *App) GetStoreConfig() map[string]string {
	if a.config == nil || a.config.StoreIdx == "" {
		return nil
	}
	return map[string]string{
		"idx":  a.config.StoreIdx,
		"name": a.config.StoreName,
	}
}

// --- Auth (Device Code Flow) ---

func (a *App) Login() (*DeviceCode, error) {
	dc, err := a.msalApp.AcquireTokenByDeviceCode(context.Background(), msalScopes)
	if err != nil {
		return nil, fmt.Errorf("디바이스 코드 생성 실패: %w", err)
	}
	a.pendingDC = &dc
	return &DeviceCode{
		UserCode:        dc.Result.UserCode,
		VerificationURL: dc.Result.VerificationURL,
	}, nil
}

func (a *App) WaitForLogin() error {
	if a.pendingDC == nil {
		return fmt.Errorf("로그인이 시작되지 않았습니다")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	result, err := a.pendingDC.AuthenticationResult(ctx)
	a.pendingDC = nil
	if err != nil {
		return fmt.Errorf("인증 실패: %w", err)
	}
	a.account = result.Account
	a.loggedIn = true
	return nil
}

func (a *App) IsLoggedIn() bool {
	return a.loggedIn
}

func (a *App) Logout() error {
	a.loggedIn = false
	a.account = public.Account{}
	os.Remove(a.configPath("token_cache.json"))

	a.mu.Lock()
	if a.stopCh != nil {
		close(a.stopCh)
		a.stopCh = nil
	}
	if a.config != nil {
		a.config.Targets = nil
		a.config.Times = nil
	}
	a.mu.Unlock()
	a.saveConfig()

	return nil
}

// --- Chat API ---

func (a *App) GetAllTargets() ([]SendTarget, error) {
	token, err := a.getToken()
	if err != nil {
		return nil, err
	}

	meBody, _ := a.graphGet("/me", token)
	var me struct {
		ID          string `json:"id"`
		DisplayName string `json:"displayName"`
	}
	json.Unmarshal(meBody, &me)

	type chatRaw struct {
		ID       string `json:"id"`
		Topic    string `json:"topic"`
		ChatType string `json:"chatType"`
		Members  []struct {
			DisplayName string `json:"displayName"`
			UserID      string `json:"userId"`
		} `json:"members"`
	}

	body, err := a.graphGet("/me/chats?$expand=members&$top=50", token)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Value []chatRaw `json:"value"`
	}
	json.Unmarshal(body, &resp)

	var targets []SendTarget
	selfFound := false
	var oneOnOne, groupChats []SendTarget

	for _, c := range resp.Value {
		name := c.Topic
		if name == "" {
			var names []string
			for _, m := range c.Members {
				if m.DisplayName != "" {
					names = append(names, m.DisplayName)
				}
			}
			name = strings.Join(names, ", ")
		}
		if name == "" {
			name = "(이름 없음)"
		}

		isSelf := len(c.Members) > 0
		for _, m := range c.Members {
			if m.UserID != me.ID {
				isSelf = false
				break
			}
		}
		if isSelf && !selfFound {
			selfFound = true
			selfLabel := me.DisplayName + " (나)"
			if me.DisplayName == "" {
				selfLabel = "나에게 보내기"
			}
			targets = append(targets, SendTarget{ID: c.ID, Name: selfLabel, Type: "chat", Group: ""})
			continue
		}

		if c.ChatType == "group" || c.ChatType == "meeting" {
			groupChats = append(groupChats, SendTarget{ID: c.ID, Name: name, Type: "chat", Group: "그룹 채팅"})
		} else {
			oneOnOne = append(oneOnOne, SendTarget{ID: c.ID, Name: name, Type: "chat", Group: "채팅"})
		}
	}

	if !selfFound {
		selfLabel := "나에게 보내기"
		if me.DisplayName != "" {
			selfLabel = me.DisplayName + " (나)"
		}
		targets = append(targets, SendTarget{ID: "__self__", Name: selfLabel, Type: "self", Group: ""})
	}

	targets = append(targets, groupChats...)
	targets = append(targets, oneOnOne...)

	return targets, nil
}

func (a *App) SendMealToChat(chatId string) (string, error) {
	a.addLog("INFO", "SEND", chatId, 0, "전송 시작")
	token, err := a.getToken()
	if err != nil {
		a.addLog("ERROR", "AUTH", chatId, 0, "토큰 획득 실패: "+err.Error())
		return "", err
	}
	a.addLog("INFO", "SEND", chatId, 0, "토큰 획득 완료")
	if chatId == "__self__" {
		realId, err := a.getOrCreateSelfChat(token)
		if err != nil {
			a.addLog("ERROR", "SELF_CHAT", chatId, 0, "본인 채팅 생성 실패: "+err.Error())
			return "", fmt.Errorf("본인 채팅 생성 실패: %w", err)
		}
		chatId = realId
	}

	mealData, err := a.GetTodayMeal()
	if err != nil {
		a.addLog("ERROR", "MEAL_API", chatId, 0, "식단 조회 실패: "+err.Error())
		return "", fmt.Errorf("식단 조회 실패: %w", err)
	}
	card := a.buildAdaptiveCard(mealData)
	payload, _ := json.Marshal(map[string]interface{}{
		"body": map[string]interface{}{
			"contentType": "html",
			"content":     `<attachment id="meal-card"></attachment>`,
		},
		"attachments": []map[string]interface{}{
			{
				"id":          "meal-card",
				"contentType": "application/vnd.microsoft.card.adaptive",
				"contentUrl":  nil,
				"content":     card,
			},
		},
	})

	today := time.Now().Format("2006-01-02")
	a.mu.Lock()
	rec, hasRecord := a.sentRecords[chatId]
	a.mu.Unlock()

	if hasRecord && rec.Date == today {
		a.deleteChatMessage(chatId, rec.MessageID, token)
	}

	url := fmt.Sprintf("%s/chats/%s/messages", graphBaseURL, chatId)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		a.addLog("ERROR", "POST", chatId, 0, "전송 실패: "+err.Error())
		return "", err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 201 {
		a.addLog("ERROR", "POST", chatId, resp.StatusCode, "전송 실패: "+string(respBody))
		return "", fmt.Errorf("전송 실패 (%d): %s", resp.StatusCode, string(respBody))
	}
	var msgResp struct {
		ID string `json:"id"`
	}
	json.Unmarshal(respBody, &msgResp)

	a.mu.Lock()
	a.sentRecords[chatId] = &sentRecord{MessageID: msgResp.ID, Date: today}
	a.mu.Unlock()
	a.saveSentRecords()

	a.addLog("INFO", "POST", chatId, 201, "전송 성공: msgId="+msgResp.ID)
	return msgResp.ID, nil
}

func (a *App) deleteChatMessage(chatId, messageId, token string) {
	meBody, err := a.graphGet("/me", token)
	if err != nil {
		a.addLog("WARN", "DELETE", chatId, 0, "사용자 ID 조회 실패: "+err.Error())
		return
	}
	var me struct {
		ID string `json:"id"`
	}
	json.Unmarshal(meBody, &me)

	url := fmt.Sprintf("%s/users/%s/chats/%s/messages/%s/softDelete", graphBaseURL, me.ID, chatId, messageId)
	req, _ := http.NewRequest("POST", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		a.addLog("WARN", "DELETE", chatId, 0, fmt.Sprintf("삭제 요청 실패: msgId=%s err=%s", messageId, err.Error()))
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == 204 || resp.StatusCode == 200 {
		a.addLog("INFO", "DELETE", chatId, resp.StatusCode, fmt.Sprintf("삭제 성공: msgId=%s", messageId))
	} else {
		a.addLog("WARN", "DELETE", chatId, resp.StatusCode, fmt.Sprintf("삭제 실패: msgId=%s body=%s", messageId, string(body)))
	}
}

// --- Adaptive Card builder ---

func (a *App) buildAdaptiveCard(data map[string]interface{}) string {
	now := time.Now()
	hour := now.Hour()
	dateStr := fmt.Sprintf("%d/%d/%d", now.Year(), int(now.Month()), now.Day())
	mealNames := map[string]string{"1": "조식", "2": "중식", "3": "석식"}
	mealEmojis := map[string]string{"1": "🌅", "2": "☀️", "3": "🌙"}

	var bodyItems []interface{}
	bodyItems = append(bodyItems, map[string]interface{}{
		"type": "TextBlock", "text": fmt.Sprintf("🍽️ 오늘의 식단 (%s)", dateStr),
		"size": "Large", "weight": "Bolder",
	})

	for _, code := range []string{"1", "2", "3"} {
		meals, ok := data[code].([]interface{})
		if !ok || len(meals) == 0 {
			continue
		}

		isPast := hour >= mealEndHours[code]
		contentId := fmt.Sprintf("meal-%s", code)
		arrowDownId := fmt.Sprintf("arrow-down-%s", code)
		arrowUpId := fmt.Sprintf("arrow-up-%s", code)

		headerSize := "Large"
		headerSubtle := false
		if isPast {
			headerSize = "Medium"
			headerSubtle = true
		}

		bodyItems = append(bodyItems, map[string]interface{}{
			"type": "ColumnSet", "spacing": "Large",
			"selectAction": map[string]interface{}{
				"type":           "Action.ToggleVisibility",
				"targetElements": []string{contentId, arrowDownId, arrowUpId},
			},
			"columns": []interface{}{
				map[string]interface{}{
					"type": "Column", "width": "stretch",
					"items": []interface{}{map[string]interface{}{
						"type": "TextBlock", "text": fmt.Sprintf("%s %s", mealEmojis[code], mealNames[code]),
						"size": headerSize, "weight": "Bolder", "isSubtle": headerSubtle,
					}},
				},
				map[string]interface{}{
					"type": "Column", "width": "auto", "verticalContentAlignment": "Center",
					"items": []interface{}{
						map[string]interface{}{"type": "TextBlock", "text": "⌄", "isSubtle": true, "id": arrowDownId, "isVisible": isPast, "horizontalAlignment": "Center"},
						map[string]interface{}{"type": "TextBlock", "text": "⌃", "isSubtle": true, "id": arrowUpId, "isVisible": !isPast, "horizontalAlignment": "Center"},
					},
				},
			},
		})

		var contentItems []interface{}
		for _, m := range meals {
			meal, ok := m.(map[string]interface{})
			if !ok {
				continue
			}
			name, _ := meal["name"].(string)
			corner, _ := meal["corner"].(string)
			side, _ := meal["side"].(string)
			kcalVal, _ := meal["kcal"].(float64)
			thumb, _ := meal["thumbnailUrl"].(string)

			if thumb != "" {
				contentItems = append(contentItems, map[string]interface{}{
					"type": "Image", "url": thumb, "size": "Stretch",
					"selectAction": map[string]interface{}{
						"type": "Action.OpenUrl",
						"url":  thumb,
					},
				})
			}
			if corner != "" {
				contentItems = append(contentItems, map[string]interface{}{"type": "TextBlock", "text": fmt.Sprintf("**[%s]**", corner), "wrap": true, "size": "Medium"})
			}
			nameCol := map[string]interface{}{
				"type": "Column", "width": "stretch",
				"items": []interface{}{map[string]interface{}{"type": "TextBlock", "text": name, "wrap": true, "size": "Medium"}},
			}
			columns := []interface{}{nameCol}
			if kcalVal > 0 {
				columns = append(columns, map[string]interface{}{
					"type": "Column", "width": "auto", "verticalContentAlignment": "Center",
					"items": []interface{}{map[string]interface{}{"type": "TextBlock", "text": fmt.Sprintf("📊 %.0f kcal", kcalVal), "isSubtle": true, "size": "Small", "horizontalAlignment": "Right"}},
				})
			}
			contentItems = append(contentItems, map[string]interface{}{"type": "ColumnSet", "spacing": "None", "columns": columns})
			if side != "" {
				contentItems = append(contentItems, map[string]interface{}{"type": "TextBlock", "text": side, "isSubtle": true, "wrap": true, "size": "Small", "spacing": "Small"})
			}
		}

		bodyItems = append(bodyItems, map[string]interface{}{
			"type": "Container", "id": contentId,
			"isVisible": !isPast,
			"items":     contentItems,
		})
	}

	card := map[string]interface{}{
		"type": "AdaptiveCard", "$schema": "http://adaptivecards.io/schemas/adaptive-card.json",
		"version": "1.5", "body": bodyItems,
		"msTeams": map[string]interface{}{"width": "Full"},
	}
	cardJSON, _ := json.Marshal(card)
	return string(cardJSON)
}

// --- Scheduler ---

func (a *App) SetSchedule(targetsJSON, times string) error {
	var targets []ConfigTarget
	if err := json.Unmarshal([]byte(targetsJSON), &targets); err != nil {
		return fmt.Errorf("대상 파싱 실패: %w", err)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.config == nil {
		a.config = &AppConfig{}
	}
	a.config.Targets = targets
	a.config.Times = strings.Split(times, ",")
	if err := a.saveConfig(); err != nil {
		return err
	}
	if a.stopCh != nil {
		close(a.stopCh)
	}
	a.startScheduler()
	return nil
}

func (a *App) GetSchedule() *Schedule {
	if a.config == nil {
		return nil
	}
	return &Schedule{Targets: a.config.Targets, Times: a.config.Times}
}

func (a *App) GetSavedConfig() *AppConfig {
	return a.config
}

func (a *App) startScheduler() {
	if a.config == nil || len(a.config.Times) == 0 || len(a.config.Targets) == 0 {
		return
	}
	a.stopCh = make(chan struct{})

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		sentToday := make(map[string]bool)
		var lastDate string

		for {
			select {
			case <-ticker.C:
				now := time.Now()
				today := now.Format("2006-01-02")
				if today != lastDate {
					sentToday = make(map[string]bool)
					lastDate = today
				}
				nowTime := now.Format("15:04")
				for _, t := range a.config.Times {
					t = strings.TrimSpace(t)
					if t == nowTime && !sentToday[t] {
						sentToday[t] = true
						if a.loggedIn {
							a.addLog("INFO", "SCHEDULE", "", 0, fmt.Sprintf("스케줄 전송 시작: %s, 대상 %d건", t, len(a.config.Targets)))
							for _, tgt := range a.config.Targets {
								if _, err := a.SendMealToChat(tgt.ID); err != nil {
									a.addLog("ERROR", "SCHEDULE", tgt.Name, 0, "스케줄 전송 실패: "+err.Error())
								}
							}
						} else {
							a.addLog("WARN", "SCHEDULE", "", 0, "스케줄 전송 스킵: 로그인 안 됨")
						}
					}
				}
			case <-a.stopCh:
				return
			}
		}
	}()
}

// --- Token ---

func (a *App) getToken() (string, error) {
	if !a.loggedIn {
		return "", fmt.Errorf("로그인이 필요합니다")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	result, err := a.msalApp.AcquireTokenSilent(ctx, msalScopes,
		public.WithSilentAccount(a.account),
	)
	if err != nil {
		a.loggedIn = false
		return "", fmt.Errorf("토큰 갱신 실패: %w", err)
	}
	return result.AccessToken, nil
}

func (a *App) getOrCreateSelfChat(token string) (string, error) {
	meBody, err := a.graphGet("/me", token)
	if err != nil {
		return "", err
	}
	var me struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(meBody, &me); err != nil {
		return "", err
	}

	nextURL := "/me/chats?$expand=members&$top=50"
	for nextURL != "" {
		body, err := a.graphGet(nextURL, token)
		if err != nil {
			return "", err
		}
		var resp struct {
			Value []struct {
				ID       string `json:"id"`
				ChatType string `json:"chatType"`
				Members  []struct {
					UserID string `json:"userId"`
				} `json:"members"`
			} `json:"value"`
			NextLink string `json:"@odata.nextLink"`
		}
		if err := json.Unmarshal(body, &resp); err != nil {
			return "", err
		}
		for _, c := range resp.Value {
			if c.ChatType != "oneOnOne" {
				continue
			}
			allMe := true
			for _, m := range c.Members {
				if m.UserID != me.ID {
					allMe = false
					break
				}
			}
			if allMe && len(c.Members) > 0 {
				return c.ID, nil
			}
		}
		nextURL = ""
		if resp.NextLink != "" {
			nextURL = strings.TrimPrefix(resp.NextLink, graphBaseURL)
		}
	}
	return "", fmt.Errorf("본인 채팅을 찾을 수 없습니다. Teams에서 먼저 자신에게 메시지를 보내주세요")
}

func (a *App) DebugChats() (string, error) {
	token, err := a.getToken()
	if err != nil {
		return "", err
	}
	meBody, _ := a.graphGet("/me", token)
	body, err := a.graphGet("/me/chats?$expand=members&$top=10", token)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("ME: %s\n\nCHATS: %s", string(meBody), string(body)), nil
}

func (a *App) GetLogFile() (string, error) {
	data, err := os.ReadFile(a.configPath("app.log"))
	if err != nil {
		return "", nil
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) > 200 {
		lines = lines[len(lines)-200:]
	}
	return strings.Join(lines, "\n"), nil
}

func (a *App) graphGet(path, token string) ([]byte, error) {
	req, _ := http.NewRequest("GET", graphBaseURL+path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// --- Config persistence ---

func (a *App) configDir() string {
	dir := filepath.Join(os.Getenv("LOCALAPPDATA"), appTitle)
	os.MkdirAll(dir, 0755)
	return dir
}

func (a *App) configPath(name string) string {
	return filepath.Join(a.configDir(), name)
}

func (a *App) loadConfig() {
	data, err := os.ReadFile(a.configPath("config.json"))
	if err != nil {
		a.addLog("INFO", "CONFIG", "", 0, "config.json 없음 (최초 실행)")
		return
	}
	var cfg AppConfig
	if json.Unmarshal(data, &cfg) != nil {
		a.addLog("WARN", "CONFIG", "", 0, "config.json 파싱 실패")
		return
	}

	if len(cfg.Targets) == 0 {
		var old struct {
			ChatID      string `json:"chatId"`
			TargetName  string `json:"targetName"`
			TargetType  string `json:"targetType"`
			TargetGroup string `json:"targetGroup"`
		}
		json.Unmarshal(data, &old)
		if old.TargetName != "" && old.ChatID != "" {
			cfg.Targets = []ConfigTarget{{
				ID:    old.ChatID,
				Name:  old.TargetName,
				Type:  old.TargetType,
				Group: old.TargetGroup,
			}}
			a.addLog("INFO", "CONFIG", "", 0, "레거시 형식에서 마이그레이션됨")
		}
	}

	a.config = &cfg
	a.addLog("INFO", "CONFIG", "", 0, fmt.Sprintf("config 로드: store=%s, targets=%d, times=%d", cfg.StoreIdx, len(cfg.Targets), len(cfg.Times)))
}

func (a *App) saveConfig() error {
	data, err := json.MarshalIndent(a.config, "", "  ")
	if err != nil {
		a.addLog("ERROR", "CONFIG", "", 0, "config 직렬화 실패: "+err.Error())
		return err
	}
	if err := os.WriteFile(a.configPath("config.json"), data, 0644); err != nil {
		a.addLog("ERROR", "CONFIG", "", 0, "config 저장 실패: "+err.Error())
		return err
	}
	a.addLog("INFO", "CONFIG", "", 0, fmt.Sprintf("config 저장: targets=%d, times=%d", len(a.config.Targets), len(a.config.Times)))
	return nil
}

func (a *App) ResetAll() error {
	a.mu.Lock()
	if a.stopCh != nil {
		close(a.stopCh)
		a.stopCh = nil
	}
	a.config = nil
	a.sentRecords = make(map[string]*sentRecord)
	a.mu.Unlock()

	a.logMu.Lock()
	a.logs = make([]LogEntry, 0)
	a.logMu.Unlock()

	a.loggedIn = false
	a.account = public.Account{}

	os.Remove(a.configPath("config.json"))
	os.Remove(a.configPath("token_cache.json"))
	os.Remove(a.configPath("sent_records.json"))
	os.Remove(a.configPath("app.log"))
	a.SetAutoStart(false)

	return nil
}

// --- Autostart (Registry) ---

const autoStartKey = `Software\Microsoft\Windows\CurrentVersion\Run`
var autoStartName = appTitle

func (a *App) GetAutoStart() bool {
	key, err := registry.OpenKey(registry.CURRENT_USER, autoStartKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer key.Close()
	_, _, err = key.GetStringValue(autoStartName)
	return err == nil
}

func (a *App) SetAutoStart(enabled bool) error {
	if enabled {
		exePath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("실행 파일 경로 확인 실패: %w", err)
		}
		key, _, err := registry.CreateKey(registry.CURRENT_USER, autoStartKey, registry.SET_VALUE)
		if err != nil {
			return fmt.Errorf("레지스트리 키 열기 실패: %w", err)
		}
		defer key.Close()
		return key.SetStringValue(autoStartName, exePath)
	}
	key, err := registry.OpenKey(registry.CURRENT_USER, autoStartKey, registry.SET_VALUE)
	if err != nil {
		return nil
	}
	defer key.Close()
	key.DeleteValue(autoStartName)
	return nil
}

func (a *App) fixAutoStartPath() {
	key, err := registry.OpenKey(registry.CURRENT_USER, autoStartKey, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return
	}
	defer key.Close()
	regPath, _, err := key.GetStringValue(autoStartName)
	if err != nil {
		return
	}
	exePath, err := os.Executable()
	if err != nil {
		return
	}
	if !strings.EqualFold(regPath, exePath) {
		key.SetStringValue(autoStartName, exePath)
		a.addLog("INFO", "AUTOSTART", "", 0, fmt.Sprintf("경로 갱신: %s → %s", regPath, exePath))
	}
}

// --- MSAL file-based token cache ---

type fileCache struct {
	file string
	mu   sync.Mutex
}

func (c *fileCache) Replace(ctx context.Context, cache msalcache.Unmarshaler, hints msalcache.ReplaceHints) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	data, err := os.ReadFile(c.file)
	if err != nil {
		return nil
	}
	return cache.Unmarshal(data)
}

func (c *fileCache) Export(ctx context.Context, cache msalcache.Marshaler, hints msalcache.ExportHints) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	data, err := cache.Marshal()
	if err != nil {
		return err
	}
	dir := filepath.Dir(c.file)
	os.MkdirAll(dir, 0700)
	return os.WriteFile(c.file, data, 0600)
}
