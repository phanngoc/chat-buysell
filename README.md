# Chat App for Buying and Selling

## Overview
This application allows users to post buying and selling requests through chat messages. The system leverages Natural Language Processing (NLP) to classify messages and match buyers with sellers. Users can continue chatting to finalize their transactions. Additionally, the system sends SMS notifications to the topic owner and push notifications via Firebase Cloud Messaging (FCM) with shortcuts like `@{username}`.

## Target Audience
- Buyers and sellers looking for a streamlined platform to connect and transact.

## Key Features
- **No Forms**: Users post buying or selling requests directly through chat messages, making the process quick and lightweight.
- **Automated Classification and Matching**: NLP is used to classify messages and match buyers with sellers, minimizing user effort.
- **AI-Powered Search**: Users can leverage AI agents to generate Elasticsearch queries for advanced matching.
- **Simplified Notifications**: Automatic SMS and FCM notifications ensure seamless communication with minimal steps.

## Why Choose This App?
Unlike traditional platforms like Chợ Tốt or other online marketplaces:
- No need for complex forms or category selection.
- AI-driven matching ensures efficiency and accuracy.
- Simplified notification system for better user engagement.

## Getting Started

### Prerequisites
1. Install [Go](https://golang.org/) and [MongoDB](https://www.mongodb.com/).
2. Ensure MongoDB is running at `mongodb://localhost:27017`.
3. Register a Facebook App to obtain the `Client ID` and `Client Secret`.

### Setup
1. Create a `.env` file or export the following environment variables:
   ```
   export FACEBOOK_CLIENT_ID=your_facebook_client_id
   export FACEBOOK_CLIENT_SECRET=your_facebook_client_secret
   ```

2. Install dependencies:
   ```
   go mod tidy
   ```

3. Run the server:
   ```
   go run main.go
   ```

4. Access the application at:
   - `http://localhost:8080/auth/facebook` to log in via Facebook.

### MongoDB Schema
- **Database**: `chatbuysell`
- **Collection**: `users`
- **Sample Document**:
  ```json
  {
    "uid": "facebook_id",
    "email": "user@email.com",
    "avatar": "https://...",
    "accessToken": "..."
  }
  ```

## API Endpoints
- `GET /auth/facebook`: Redirects to Facebook login.
- `GET /auth/facebook/callback`: Handles the callback and returns user information.

## Project Structure
- `main.go`: Entry point of the application.
- `models.go`: Contains data models.
- `docker-compose.yml`: Configuration for Docker.
- `go.mod` and `go.sum`: Dependency management files.
- `README.md`: Project documentation.
- `roadmap.md`: Future development plans.

## Roadmap
- Implement advanced AI-based matching algorithms.
- Add support for additional notification channels.
- Enhance the user interface for better usability.

## Contributing
Contributions are welcome! Please fork the repository and submit a pull request.

## License
This project is licensed under the MIT License. See the `LICENSE` file for details.
