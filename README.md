# LivePoll

A real-time polling application where users can create polls, vote on options, and see vote counts update live without refreshing the page.

## 🚀 Live Demo

**Live Application:**  
https://live-poll-fxjxvsdgs-fatima-f9f7.vercel.app

**GitHub Repository:**  
https://github.com/fatima064/LivePoll

---

## 📌 About the Project

LivePoll is a full-stack real-time polling application built as an internship assignment.

The application allows users to:

- Create an account and log in
- Create polls with multiple options
- Vote on poll options
- View current vote counts
- Receive live vote count updates without refreshing the page
- Access the application through a deployed public URL

The main focus of the project is implementing real-time updates using **Redis Pub/Sub and Server-Sent Events (SSE)**.

---

## ✨ Features

### Authentication
- User signup and login
- Password hashing
- JWT-based authentication
- Protected poll creation endpoint

### Polls
- Create polls with multiple options
- View poll questions and options
- Vote on available options
- View current vote counts

### Real-Time Updates
- Redis stores live vote counters
- Redis Pub/Sub broadcasts vote updates
- Server-Sent Events (SSE) sends updates from the backend to connected browsers
- Vote counts update automatically without refreshing the page

### Deployment
- Frontend deployed using Vercel
- Backend deployed using Render
- MongoDB hosted using MongoDB Atlas
- Redis hosted using Redis Cloud

---

## 🛠️ Tech Stack

### Frontend
- React
- Vite
- JavaScript
- CSS
- Server-Sent Events (SSE)

### Backend
- Go
- Gin Framework
- JWT Authentication

### Database & Real-Time Services
- MongoDB Atlas
- Redis Cloud
- Redis Pub/Sub

### Deployment
- Vercel
- Render
- GitHub

---

## 🔄 How Real-Time Voting Works
```text
The real-time voting flow works as follows:

User votes
    ↓
React Frontend
    ↓
Go + Gin Backend
    ↓
Redis
    ├── Updates vote counter
    └── Publishes vote event
            ↓
       Redis Pub/Sub
            ↓
       SSE Connection
            ↓
     Connected Browsers
            ↓
    Vote count updates
    without page refresh
