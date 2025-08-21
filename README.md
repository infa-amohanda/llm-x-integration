# History News Bot

An automated Go application that uses Google Gemini AI to generate Liverpool FC news content and posts it to X.com (Twitter) every 4 hours via GitHub Actions.

## Features

- 🤖 AI-powered content generation using Google Gemini API
- 🐦 Automated posting to X.com (Twitter)
- ⏰ Scheduled execution every 4 hours via GitHub Actions
- 🔧 Configurable prompts and settings
- 🔒 Secure credential management via environment variables

## Prerequisites

1. **Google AI API Key**: Get your API key from [Google AI Studio](https://makersuite.google.com/app/apikey)
2. **Twitter API Credentials**: Apply for Twitter API access at [developer.twitter.com](https://developer.twitter.com/)
   - Consumer Key
   - Consumer Secret
   - Access Token
   - Access Token Secret

## Local Setup

1. **Clone the repository**:
   ```bash
   git clone <your-repo-url>
   cd llm-x-integration
   ```

2. **Install Go dependencies**:
   ```bash
   go mod tidy
   ```

3. **Set up environment variables**:
   ```bash
   cp .env.example .env
   # Edit .env with your actual API credentials
   ```

4. **Test the application**:
   ```bash
   go run main.go
   ```

## GitHub Actions Setup

1. **Add Repository Secrets**:
   Go to your GitHub repository → Settings → Secrets and variables → Actions

   Add the following secrets:
   - `GOOGLE_API_KEY`: Your Google Gemini API key
   - `TWITTER_CONSUMER_KEY`: Your Twitter consumer key
   - `TWITTER_CONSUMER_SECRET`: Your Twitter consumer secret
   - `TWITTER_ACCESS_TOKEN`: Your Twitter access token
   - `TWITTER_ACCESS_TOKEN_SECRET`: Your Twitter access token secret
   - `LIVERPOOL_NEWS_PROMPT` (optional): Custom prompt for content generation

2. **Enable GitHub Actions**:
   The workflow will automatically run every 4 hours once you push the code to your repository.

3. **Manual Trigger**:
   You can manually trigger the workflow from the Actions tab in your GitHub repository.

## Configuration

### Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `GOOGLE_API_KEY` | Yes | Google Gemini API key |
| `TWITTER_CONSUMER_KEY` | Yes | Twitter API consumer key |
| `TWITTER_CONSUMER_SECRET` | Yes | Twitter API consumer secret |
| `TWITTER_ACCESS_TOKEN` | Yes | Twitter API access token |
| `TWITTER_ACCESS_TOKEN_SECRET` | Yes | Twitter API access token secret |
| `LIVERPOOL_NEWS_PROMPT` | No | Custom prompt for content generation |

### Default Prompt

If `LIVERPOOL_NEWS_PROMPT` is not set, the bot uses this default prompt:

```
Generate a concise and engaging tweet about Liverpool FC news. Focus on recent matches, transfers, or club updates. Keep it under 280 characters and make it engaging for football fans. Include relevant hashtags like #LFC #Liverpool
```

## Project Structure

```
.
├── main.go                           # Main application code
├── go.mod                           # Go module dependencies
├── .env.example                     # Environment variables template
├── .github/workflows/
│   └── liverpool-news-bot.yml      # GitHub Actions workflow
└── README.md                        # This file
```

## How It Works

1. **Content Generation**: The bot uses Google Gemini AI to generate Liverpool FC-related content based on the configured prompt
2. **Content Validation**: Ensures the content is within Twitter's 280-character limit
3. **Posting**: Posts the generated content to X.com using the Twitter API
4. **Scheduling**: GitHub Actions runs the bot every 4 hours automatically

## Customization

You can customize the bot by:

1. **Modifying the prompt**: Change the `LIVERPOOL_NEWS_PROMPT` environment variable
2. **Adjusting the schedule**: Edit the cron expression in `.github/workflows/liverpool-news-bot.yml`
3. **Adding more features**: Extend the Go code to include images, multiple posts, or different content types

## Troubleshooting

### Common Issues

1. **API Rate Limits**: Both Google Gemini and Twitter have rate limits. The 4-hour schedule helps avoid these.

2. **Invalid Credentials**: Ensure all API credentials are correctly set in GitHub Secrets.

3. **Content Generation Fails**: Check that your Google API key has access to the Gemini API.

4. **Twitter Posting Fails**: Verify your Twitter API credentials and that your app has write permissions.

### Logs

Check the GitHub Actions logs for detailed error messages and execution details.

## License

This project is open source and available under the [MIT License](LICENSE).

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test locally
5. Submit a pull request

## Disclaimer

This bot is for educational and personal use. Ensure compliance with Twitter's Terms of Service and API usage policies when using this bot.
