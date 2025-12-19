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
	FootballDataAPIKey  string
	NewsAPIKey          string // NEW
	PerplexityAPIKey    string // NEW
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

type NewsAPIArticle struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Url         string `json:"url"`
	Source      struct {
		Name string `json:"name"`
	} `json:"source"`
}

type NewsAPIResponse struct {
	Status       string           `json:"status"`
	TotalResults int              `json:"totalResults"`
	Articles     []NewsAPIArticle `json:"articles"`
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
		NewsAPIKey:          os.Getenv("NEWS_API_KEY"),          // NEW
		PerplexityAPIKey:    os.Getenv("PERPLEXITY_API_KEY"),    // NEW
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
	if config.NewsAPIKey == "" {
		return nil, fmt.Errorf("NEWS_API_KEY is required")
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
	model := nb.geminiClient.GenerativeModel("gemini-flash-latest")
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
	model := nb.geminiClient.GenerativeModel("gemini-flash-latest")
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

func (nb *NewsBot) fetchPerplexityCryptoTweet(ctx context.Context, article *NewsAPIArticle) (string, error) {
	if nb.config.PerplexityAPIKey == "" {
		return "", fmt.Errorf("Perplexity API key not set")
	}
	url := "https://api.perplexity.ai/chat/completions"
	payload := map[string]interface{}{
		"model": "sonar-pro",
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": "You are an expert crypto Twitter writer. Write engaging, informative tweets with emojis where appropriate. Always include relevant hashtags like #Crypto #Blockchain #CryptoNews. Keep tweets under 280 characters.",
			},
			{
				"role":    "user",
				"content": fmt.Sprintf("Generate a tweet about this crypto news headline and summary.\nThe tweet must be at least 100 characters long, under 280 characters, engaging and informative.\nInclude hashtags like #Crypto #Blockchain #CryptoNews.\n\nTitle: %s\nDescription: %s\nSource: %s", article.Title, article.Description, article.Source.Name),
			},
		},
		"max_tokens":  500,
		"temperature": 0.8,
		"top_p":       0.9,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %v", err)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("accept", "application/json")
	req.Header.Set("content-type", "application/json")
	req.Header.Set("Authorization", "Bearer "+nb.config.PerplexityAPIKey)
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call Perplexity API: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Perplexity API error: %s", string(body))
	}
	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode Perplexity response: %v", err)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no choices returned from Perplexity")
	}
	content := strings.TrimSpace(result.Choices[0].Message.Content)
	if len(content) > 280 {
		content = content[:277] + "..."
	}
	return content, nil
}

func (nb *NewsBot) fetchPerplexityFootballTweet(ctx context.Context, leagueName string, match *PremierLeagueMatch) (string, error) {
	if nb.config.PerplexityAPIKey == "" {
		return "", fmt.Errorf("Perplexity API key not set")
	}
	url := "https://api.perplexity.ai/chat/completions"
	prompt := fmt.Sprintf("You are an expert football Twitter writer. Write engaging, informative tweets with emojis where appropriate. Always include relevant hashtags like #%s #Football #FootballNews. Keep tweets under 280 characters. Generate a tweet about the latest %s football result. The tweet must be at least 100 characters long, under 280 characters, engaging and informative.\nMatch: %s %d - %d %s\nDate: %s", leagueName, leagueName, match.HomeTeam.Name, match.Score.FullTime.Home, match.Score.FullTime.Away, match.AwayTeam.Name, match.UtcDate[:10])
	payload := map[string]interface{}{
		"model": "sonar-pro",
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": fmt.Sprintf("You are an expert football Twitter writer. Write engaging, informative tweets with emojis where appropriate. Always include relevant hashtags like #%s #Football #FootballNews. Keep tweets under 280 characters.", leagueName),
			},
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"max_tokens":  500,
		"temperature": 0.8,
		"top_p":       0.9,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %v", err)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("accept", "application/json")
	req.Header.Set("content-type", "application/json")
	req.Header.Set("Authorization", "Bearer "+nb.config.PerplexityAPIKey)
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call Perplexity API: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Perplexity API error: %s", string(body))
	}
	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode Perplexity response: %v", err)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no choices returned from Perplexity")
	}
	content := strings.TrimSpace(result.Choices[0].Message.Content)
	if len(content) > 280 {
		content = content[:277] + "..."
	}
	return content, nil
}

func (nb *NewsBot) generateCryptoNewsFromAPI(ctx context.Context) (string, error) {
	article, err := nb.fetchLatestCryptoNews(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to fetch crypto news: %v", err)
	}
	prompt := fmt.Sprintf(`Generate a tweet about this crypto news headline and summary.\nTitle: %s\nDescription: %s\nSource: %s\nRequirements:\n- The tweet must be at least 100 characters long.\n- Keep it under 280 characters.\n- Make it engaging and informative.\n- Include hashtags like #Crypto #Blockchain #News.`,
		article.Title, article.Description, article.Source.Name)
	model := nb.geminiClient.GenerativeModel("gemini-flash-latest")
	model.SetTemperature(0.7)
	model.SetMaxOutputTokens(200)
	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		log.Println("Gemini API failed, using Perplexity fallback for crypto...")
		return nb.fetchPerplexityCryptoTweet(ctx, article)
	}
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		log.Println("Gemini API returned no content, using Perplexity fallback for crypto...")
		return nb.fetchPerplexityCryptoTweet(ctx, article)
	}
	content := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])
	content = strings.TrimSpace(content)
	content = strings.Trim(content, "\"")
	if len(content) > 280 {
		content = content[:277] + "..."
	}
	return content, nil
}

func (nb *NewsBot) fetchLatestCryptoNews(ctx context.Context) (*NewsAPIArticle, error) {
	url := "https://newsapi.org/v2/top-headlines?q=crypto&pageSize=1"
	client := &http.Client{Timeout: 10 * time.Second}
	request, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("X-Api-Key", nb.config.NewsAPIKey)
	resp, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("newsapi.org API error: %s", string(body))
	}
	var newsResp NewsAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&newsResp); err != nil {
		return nil, err
	}
	if len(newsResp.Articles) == 0 {
		return nil, fmt.Errorf("no crypto news found")
	}
	return &newsResp.Articles[0], nil
}

func (nb *NewsBot) generatePremierLeagueNews(ctx context.Context) (string, error) {
	model := nb.geminiClient.GenerativeModel("gemini-flash-latest")
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

type FootballLeague string

const (
	PremierLeague FootballLeague = "PL"
	LaLiga        FootballLeague = "PD"
	Bundesliga    FootballLeague = "BL1"
	SerieA        FootballLeague = "SA"
	Ligue1        FootballLeague = "FL1"
	IrishPremier  FootballLeague = "IRL"
)

func (nb *NewsBot) fetchLatestLeagueMatch(ctx context.Context, league FootballLeague) (*PremierLeagueMatch, error) {
	url := fmt.Sprintf("https://api.football-data.org/v4/competitions/%s/matches?status=FINISHED&limit=5", league)
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

func (nb *NewsBot) generateLeagueNewsFromAPI(ctx context.Context, league FootballLeague, leagueName string) (string, error) {
	match, err := nb.fetchLatestLeagueMatch(ctx, league)
	if err != nil {
		return "", fmt.Errorf("failed to fetch latest match: %v", err)
	}
	date := match.UtcDate[:10] // YYYY-MM-DD
	prompt := fmt.Sprintf(`Write a complete, engaging tweet (at least 100 but under 280 characters) about the latest %s football result.\n\nMatch: %s %d - %d %s\nDate: %s\n\nMake the tweet informative and detailed, mentioning key moments or context if possible. Avoid generic statements. Include hashtags like #%s #Football. Output only the tweet text.`,
		leagueName, match.HomeTeam.Name, match.Score.FullTime.Home, match.Score.FullTime.Away, match.AwayTeam.Name, date, leagueName)
	model := nb.geminiClient.GenerativeModel("gemini-flash-latest")
	model.SetTemperature(0.8)
	model.SetMaxOutputTokens(200)
	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		log.Println("Gemini API failed, using Perplexity fallback for football...")
		return nb.fetchPerplexityFootballTweet(ctx, leagueName, match)
	}
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		log.Println("Gemini API returned no content, using Perplexity fallback for football...")
		return nb.fetchPerplexityFootballTweet(ctx, leagueName, match)
	}
	content := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])
	content = strings.TrimSpace(content)
	content = strings.Trim(content, "\"")
	if len(content) > 280 {
		content = content[:277] + "..."
	}
	if len(content) < 100 {
		// Retry with a stronger prompt if too short
		retryPrompt := fmt.Sprintf(`Write a complete, detailed tweet (at least 100 but under 280 characters) about the latest %s football result.\n\nMatch: %s %d - %d %s\nDate: %s\n\nBe detailed and informative. Mention key facts, context, and impact. Avoid generic statements. Include hashtags like #%s #Football. Output only the tweet text.`,
			leagueName, match.HomeTeam.Name, match.Score.FullTime.Home, match.Score.FullTime.Away, match.AwayTeam.Name, date, leagueName)
		resp, err = model.GenerateContent(ctx, genai.Text(retryPrompt))
		if err != nil {
			log.Println("Gemini API failed on retry, using Perplexity fallback for football...")
			return nb.fetchPerplexityFootballTweet(ctx, leagueName, match)
		}
		if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
			log.Println("Gemini API returned no content on retry, using Perplexity fallback for football...")
			return nb.fetchPerplexityFootballTweet(ctx, leagueName, match)
		}
		content = fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])
		content = strings.TrimSpace(content)
		content = strings.Trim(content, "\"")
		if len(content) > 280 {
			content = content[:277] + "..."
		}
	}
	return content, nil
}

func (nb *NewsBot) Run() error {
	ctx := context.Background()

	var content string
	var err error

	// Cycle: 0 = Premier League, 1 = La Liga, 2 = Bundesliga, 3 = Serie A, 4 = Ligue 1, 5 = Irish Premier, 6 = Crypto
	switch time.Now().Unix() % 7 {
	case 0:
		log.Println("Generating Premier League news content from API...")
		content, err = nb.generateLeagueNewsFromAPI(ctx, PremierLeague, "PremierLeague")
	case 1:
		log.Println("Generating La Liga news content from API...")
		content, err = nb.generateLeagueNewsFromAPI(ctx, LaLiga, "LaLiga")
	case 2:
		log.Println("Generating Bundesliga news content from API...")
		content, err = nb.generateLeagueNewsFromAPI(ctx, Bundesliga, "Bundesliga")
	case 3:
		log.Println("Generating Serie A news content from API...")
		content, err = nb.generateLeagueNewsFromAPI(ctx, SerieA, "SerieA")
	case 4:
		log.Println("Generating Ligue 1 news content from API...")
		content, err = nb.generateLeagueNewsFromAPI(ctx, Ligue1, "Ligue1")
	case 5:
		log.Println("Generating Irish Premier Division news content from API...")
		content, err = nb.generateLeagueNewsFromAPI(ctx, IrishPremier, "IrishPremierDivision")
	case 6:
		log.Println("Generating Crypto news content from API...")
		content, err = nb.generateCryptoNewsFromAPI(ctx)
	}

	if err != nil {
		return fmt.Errorf("failed to generate news: %v", err)
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
