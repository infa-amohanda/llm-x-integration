package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/dghubble/oauth1"
	"github.com/google/generative-ai-go/genai"
	"github.com/joho/godotenv"
	"google.golang.org/api/option"
)

type Config struct {
	GoogleAPIKey        string
	XAPIKey             string
	XAPIKeySecret       string
	XAccessToken        string
	XAccessTokenSecret  string
	LiverpoolNewsPrompt string
	FootballDataAPIKey  string // NEW
}

type NewsBot struct {
	config       *Config
	geminiClient *genai.Client
	httpClient   *http.Client
}

// X API v2 tweet request structure
type TweetRequest struct {
	Text string `json:"text"`
}

// X API v2 tweet response structure
type TweetResponse struct {
	Data struct {
		ID   string `json:"id"`
		Text string `json:"text"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"errors,omitempty"`
}

type PremierLeagueMatch struct {
	HomeTeam struct {
		Name string `json:"name"`
	} `json:"homeTeam"`
	AwayTeam struct {
		Name string `json:"name"`
	} `json:"awayTeam"`
	UtcDate string `json:"utcDate"`
	Status  string `json:"status"`
	Score   struct {
		FullTime struct {
			Home int `json:"home"`
			Away int `json:"away"`
		} `json:"fullTime"`
	} `json:"score"`
}

type PremierLeagueMatchesResponse struct {
	Matches []PremierLeagueMatch `json:"matches"`
}

func loadConfig() (*Config, error) {
	godotenv.Load()

	config := &Config{
		GoogleAPIKey:        os.Getenv("GOOGLE_API_KEY"),
		XAPIKey:             os.Getenv("X_API_KEY"),        // Changed from X_OAUTH_KEY
		XAPIKeySecret:       os.Getenv("X_API_KEY_SECRET"), // Changed from X_OAUTH_KEY_SECRET
		XAccessToken:        os.Getenv("X_ACCESS_TOKEN"),
		XAccessTokenSecret:  os.Getenv("X_ACCESS_TOKEN_SECRET"),
		LiverpoolNewsPrompt: os.Getenv("LIVERPOOL_NEWS_PROMPT"),
		FootballDataAPIKey:  os.Getenv("FOOTBALL_DATA_API_KEY"), // NEW
	}

	if config.LiverpoolNewsPrompt == "" {
		config.LiverpoolNewsPrompt = "Generate a concise and engaging tweet about Liverpool FC news. Focus on recent matches, transfers, or club updates. Keep it under 280 characters and make it engaging for football fans. Include relevant hashtags like #LFC #Liverpool"
	}

	if config.GoogleAPIKey == "" {
		return nil, fmt.Errorf("GOOGLE_API_KEY is required")
	}
	if config.XAPIKey == "" || config.XAPIKeySecret == "" ||
		config.XAccessToken == "" || config.XAccessTokenSecret == "" {
		return nil, fmt.Errorf("all X API credentials are required")
	}
	if config.FootballDataAPIKey == "" {
		return nil, fmt.Errorf("FOOTBALL_DATA_API_KEY is required")
	}

	return config, nil
}

func NewNewsBot(config *Config) (*NewsBot, error) {
	ctx := context.Background()
	geminiClient, err := genai.NewClient(ctx, option.WithAPIKey(config.GoogleAPIKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %v", err)
	}

	// Use OAuth 1.0a (revert from Bearer Token approach)
	oauthConfig := oauth1.NewConfig(config.XAPIKey, config.XAPIKeySecret)
	token := oauth1.NewToken(config.XAccessToken, config.XAccessTokenSecret)
	httpClient := oauthConfig.Client(oauth1.NoContext, token)

	return &NewsBot{
		config:       config,
		geminiClient: geminiClient,
		httpClient:   httpClient,
	}, nil
}

func (nb *NewsBot) generateLiverpoolNews(ctx context.Context) (string, error) {
	model := nb.geminiClient.GenerativeModel("gemini-2.5-flash-lite")
	model.SetTemperature(0.8) // Increased for more creativity
	model.SetMaxOutputTokens(150)

	// Get current date info for historical context
	now := time.Now()
	currentMonth := now.Format("January")
	currentDay := now.Day()

	prompt := fmt.Sprintf(`Generate an engaging tweet about Liverpool FC history for %s %d.

Focus on one of these types of historical content:
1. "On this day" historical events (matches, signings, achievements)
2. Legendary players and their memorable moments
3. Historic matches and victories
4. Club records and milestones
5. Memorable quotes from players/managers
6. Stadium history (Anfield moments)
7. European Cup/Champions League history
8. League title victories
9. FA Cup moments
10. Derby matches against Everton or Manchester United

Requirements:
- Make it feel like a "throwback" or "on this day" style post
- Include specific years, scores, or player names when possible
- Keep it under 280 characters
- Make it engaging for Liverpool fans
- Include relevant hashtags like #LFC #Liverpool #OnThisDay #YNWA
- Sound authentic and factual
- Generate only the tweet text, no quotes or formatting

Current date context: %s %d
Make it feel timely and relevant to today's date if possible.`,
		currentMonth, currentDay, currentMonth, currentDay)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %v", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no content generated")
	}

	content := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])
	content = strings.TrimSpace(content)
	content = strings.Trim(content, "\"")

	if len(content) > 280 {
		content = content[:277] + "..."
	}

	return content, nil
}

func (nb *NewsBot) generateLiverpoolHistoryVariation(ctx context.Context) (string, error) {
	model := nb.geminiClient.GenerativeModel("gemini-1.5-flash")
	model.SetTemperature(0.8)
	model.SetMaxOutputTokens(150)

	// Random historical topics
	topics := []string{
		"legendary players like Steven Gerrard, Kenny Dalglish, or Ian Rush",
		"historic European Cup victories in the 1970s and 1980s",
		"memorable Premier League moments and title wins",
		"Anfield atmosphere and famous stadium moments",
		"classic derby matches against Everton or Manchester United",
		"Bill Shankly or Bob Paisley management eras",
		"Champions League victories in 2005 and 2019",
		"FA Cup finals and memorable cup runs",
		"record-breaking performances and club milestones",
		"famous Liverpool chants and supporter culture",
	}

	// Pick a random topic
	topic := topics[time.Now().Unix()%int64(len(topics))]

	prompt := fmt.Sprintf(`Create an engaging historical tweet about Liverpool FC focusing on %s.

Make it:
- Nostalgic and celebratory
- Include specific details (years, scores, names)
- Under 280 characters
- Engaging for Liverpool supporters
- Include hashtags like #LFC #Liverpool #YNWA #History
- Sound like a passionate fan sharing a great memory

Generate only the tweet text, no formatting or quotes.`, topic)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %v", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no content generated")
	}

	content := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])
	content = strings.TrimSpace(content)
	content = strings.Trim(content, "\"")

	if len(content) > 280 {
		content = content[:277] + "..."
	}

	return content, nil
}

func (nb *NewsBot) testAuth() error {
	// Test with a simple GET request to verify auth works
	req, err := http.NewRequest("GET", "https://api.twitter.com/2/users/me", nil)
	if err != nil {
		return err
	}

	resp, err := nb.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	log.Printf("Auth test response: %d - %s", resp.StatusCode, string(body))

	return nil
}

// Add this debug function to verify credentials are loaded
func (nb *NewsBot) debugCredentials() {
	log.Printf("API Key exists: %t", nb.config.XAPIKey != "")
	log.Printf("API Key Secret exists: %t", nb.config.XAPIKeySecret != "")
	log.Printf("Access Token exists: %t", nb.config.XAccessToken != "")
	log.Printf("Access Token Secret exists: %t", nb.config.XAccessTokenSecret != "")
	log.Printf("API Key (first 8 chars): %s...", nb.config.XAPIKey[:min(8, len(nb.config.XAPIKey))])
}

func (nb *NewsBot) postToTwitter(content string) error {
	url := "https://api.twitter.com/2/tweets"
	tweetReq := TweetRequest{Text: content}

	jsonData, err := json.Marshal(tweetReq)
	if err != nil {
		return fmt.Errorf("failed to marshal tweet request: %v", err)
	}

	fmt.Print(string(jsonData))

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	// Remove any Bearer Token headers - OAuth1 client handles auth automatically

	log.Printf("Request Headers: %v", req.Header)

	resp, err := nb.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %v", err)
	}
	log.Printf("Raw Response: %s", string(body)) // Debug raw response

	var tweetResp TweetResponse
	if err := json.Unmarshal(body, &tweetResp); err != nil {
		return fmt.Errorf("failed to parse response: %v, raw response: %s", err, string(body))
	}

	if resp.StatusCode != http.StatusCreated {
		if len(tweetResp.Errors) > 0 {
			return fmt.Errorf("X API error (status %d): %s", resp.StatusCode, tweetResp.Errors[0].Message)
		}
		return fmt.Errorf("X API error (status %d): %s", resp.StatusCode, string(body))
	}

	log.Printf("Tweet posted successfully with ID: %s, Text: %s", tweetResp.Data.ID, tweetResp.Data.Text)
	return nil
}

func (nb *NewsBot) fetchLatestPremierLeagueMatch(ctx context.Context) (*PremierLeagueMatch, error) {
	url := "https://api.football-data.org/v4/competitions/PL/matches?status=FINISHED&limit=1"
	client := &http.Client{Timeout: 10 * time.Second}
	request, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("X-Auth-Token", nb.config.FootballDataAPIKey)
	request.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("football-data.org API error: %s", string(body))
	}
	var matches PremierLeagueMatchesResponse
	if err := json.NewDecoder(resp.Body).Decode(&matches); err != nil {
		return nil, err
	}
	if len(matches.Matches) == 0 {
		return nil, fmt.Errorf("no matches found")
	}
	return &matches.Matches[len(matches.Matches)-1], nil // latest finished match
}

func (nb *NewsBot) generatePremierLeagueNewsFromAPI(ctx context.Context) (string, error) {
	match, err := nb.fetchLatestPremierLeagueMatch(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to fetch latest match: %v", err)
	}
	// Format match info for Gemini
	date := match.UtcDate[:10] // YYYY-MM-DD
	prompt := fmt.Sprintf(`Generate a tweet about the latest Premier League result:\nDate: %s\n%s %d - %d %s\nMake it concise, engaging, under 280 characters, and include hashtags like #PremierLeague #EPL.`,
		date, match.HomeTeam.Name, match.Score.FullTime.Home, match.Score.FullTime.Away, match.AwayTeam.Name)
	model := nb.geminiClient.GenerativeModel("gemini-2.5-flash-lite")
	model.SetTemperature(0.7)
	model.SetMaxOutputTokens(150)
	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("failed to generate summary: %v", err)
	}
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no content generated")
	}
	content := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])
	content = strings.TrimSpace(content)
	content = strings.Trim(content, "\"")
	if len(content) > 280 {
		content = content[:277] + "..."
	}
	return content, nil
}

func (nb *NewsBot) generatePremierLeagueNews(ctx context.Context) (string, error) {
	model := nb.geminiClient.GenerativeModel("gemini-2.5-flash-lite")
	model.SetTemperature(0.7)
	model.SetMaxOutputTokens(150)

	// Get current date for context
	now := time.Now()
	currentMonth := now.Format("January")
	currentDay := now.Day()
	currentYear := now.Year()

	prompt := fmt.Sprintf(`Generate a concise, engaging tweet with the latest Premier League news as of %s %d, %d.

Requirements:
- Focus on recent matches, transfers, injuries, standings, or major headlines
- Mention specific teams, players, or results if possible
- Keep it under 280 characters
- Make it interesting for football fans
- Include relevant hashtags like #PremierLeague #EPL #Football
- Generate only the tweet text, no quotes or formatting

Current date context: %s %d, %d`,
		currentMonth, currentDay, currentYear, currentMonth, currentDay, currentYear)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("failed to generate Premier League news: %v", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no content generated")
	}

	content := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])
	content = strings.TrimSpace(content)
	content = strings.Trim(content, "\"")

	if len(content) > 280 {
		content = content[:277] + "..."
	}

	return content, nil
}

func (nb *NewsBot) Run() error {
	ctx := context.Background()

	log.Println("Generating Premier League news content from API...")

	content, err := nb.generatePremierLeagueNewsFromAPI(ctx)
	if err != nil {
		return fmt.Errorf("failed to generate Premier League news: %v", err)
	}

	log.Printf("Generated content: %s", content)

	log.Println("Posting to X...")
	err = nb.postToTwitter(content)
	if err != nil {
		return fmt.Errorf("failed to post to X: %v", err)
	}

	log.Println("Successfully posted content to X!")
	return nil
}

func (nb *NewsBot) Close() {
	if nb.geminiClient != nil {
		nb.geminiClient.Close()
	}
}

func main() {
	log.Println("Starting Liverpool News Bot...")

	config, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	bot, err := NewNewsBot(config)
	if err != nil {
		log.Fatalf("Failed to create news bot: %v", err)
	}
	defer bot.Close()

	// Add credential debugging
	bot.debugCredentials()

	if err := bot.Run(); err != nil {
		log.Fatalf("Bot execution failed: %v", err)
	}

	log.Println("Bot execution completed successfully!")
}

func (nb *NewsBot) generateIndiaHistory(ctx context.Context) (string, error) {
	model := nb.geminiClient.GenerativeModel("gemini-2.5-flash-lite")
	model.SetTemperature(0.8)
	model.SetMaxOutputTokens(150)

	// Get current date info for historical context
	now := time.Now()
	currentMonth := now.Format("January")
	currentDay := now.Day()

	prompt := fmt.Sprintf(`Generate an engaging tweet about Indian history for %s %d.

Focus on one of these types of historical content:
1. "On this day" historical events (independence movement, ancient history, battles)
2. Great Indian leaders and freedom fighters (Gandhi, Nehru, Bose, Chandragupta, Ashoka)
3. Ancient Indian achievements (science, mathematics, philosophy, architecture)
4. Cultural and religious milestones (Buddhism, Hinduism, art, literature)
5. Medieval Indian empires (Mughal, Maratha, Chola, Vijayanagara)
6. Colonial period events and resistance movements
7. Post-independence achievements (space program, technology, democracy)
8. Indian inventions and discoveries (zero, decimal system, surgery, astronomy)
9. Famous Indian monuments and their history (Taj Mahal, Red Fort, temples)
10. Indian scientists, mathematicians, and scholars throughout history

Requirements:
- Make it feel like a "throwback" or "on this day" style post
- Include specific years, names, or achievements when possible
- Keep it under 280 characters
- Make it engaging and educational
- Include relevant hashtags like #IndianHistory #India #OnThisDay #Heritage #Culture
- Sound authentic and factual
- Generate only the tweet text, no quotes or formatting

Current date context: %s %d
Make it feel timely and relevant to today's date if possible.`,
		currentMonth, currentDay, currentMonth, currentDay)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %v", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no content generated")
	}

	content := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])
	content = strings.TrimSpace(content)
	content = strings.Trim(content, "\"")

	if len(content) > 280 {
		content = content[:277] + "..."
	}

	return content, nil
}

func (nb *NewsBot) generateIndiaHistoryVariation(ctx context.Context) (string, error) {
	model := nb.geminiClient.GenerativeModel("gemini-1.5-flash")
	model.SetTemperature(0.8)
	model.SetMaxOutputTokens(150)

	// Random historical topics about India
	topics := []string{
		"ancient Indian scientific achievements and mathematicians like Aryabhata",
		"the Mughal Empire and rulers like Akbar and Shah Jahan",
		"Indian freedom fighters and the independence movement",
		"ancient Indian philosophy and spiritual leaders like Buddha",
		"the Mauryan Empire and Emperor Ashoka's reign",
		"Indian contributions to medicine and surgery in ancient times",
		"the Chola dynasty and their maritime achievements",
		"Indian art, architecture, and monument construction",
		"the Gupta period known as the Golden Age of India",
		"Indian inventions that changed the world like zero and decimal system",
		"the Maratha Empire and Shivaji's military strategies",
		"Indian classical literature and epic poems like Ramayana and Mahabharata",
		"ancient Indian universities like Nalanda and Takshashila",
		"Indian space achievements and modern technological progress",
		"the Indus Valley Civilization and Harappan culture",
	}

	// Pick a random topic
	topic := topics[time.Now().Unix()%int64(len(topics))]

	prompt := fmt.Sprintf(`Create an engaging historical tweet about India focusing on %s.

Make it:
- Educational and inspiring
- Include specific details (years, names, achievements)
- Under 280 characters
- Engaging for history enthusiasts
- Include hashtags like #IndianHistory #India #Heritage #Culture #Achievement
- Sound like sharing fascinating historical knowledge

Generate only the tweet text, no formatting or quotes.`, topic)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %v", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no content generated")
	}

	content := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])
	content = strings.TrimSpace(content)
	content = strings.Trim(content, "\"")

	if len(content) > 280 {
		content = content[:277] + "..."

	}

	return content, nil
}
