# Event-Driven Go Project 🚀

This repository contains my implementation of the Event-Driven Architecture (EDA) patterns from the Three Dots Labs course. The project demonstrates practical usage of event-driven design principles in Go.

## 🌟 Features

- Event-Driven Architecture implementation
- Message-driven communication
- Asynchronous processing
- Scalable design patterns
- Comprehensive test coverage

## 🛠 Tech Stack

- Go
- Redis (temporary solution for learning purposes)
- GitHub Actions (CI/CD)

> ⚠️ **Note on Event Store**: Redis is used in this project as a simple solution for learning EDA patterns. In production environments, consider using specialized message queues and event stores like:
> - Apache Kafka
> - RabbitMQ
> - NATS
> - Amazon SQS/SNS
> - Google Cloud Pub/Sub
>
> Choose based on your specific requirements for scalability, persistence, and message guarantees.

## 🏃‍♂️ Running Tests

Make sure you have Redis running locally on port 6379, then:

```bash
REDIS_ADDR=localhost:6379 go test ./tests/ -v
```

## 📦 CI/CD

The project is configured with GitHub Actions for automated testing. Each push and pull request triggers the test pipeline with:
- Automated Redis setup
- Go environment configuration
- Full test suite execution

## 🎓 Course Information

This project is based on the Event-Driven Go course by Three Dots Labs. It implements various EDA patterns and best practices learned during the course.

## 🏗 Project Structure

```
/
├── project/
│   ├── tests/       # Test suite
│   ├── internal/    # Internal packages
│   └── cmd/         # Application entrypoints
└── .github/
    └── workflows/   # CI/CD configurations
```

## 🤝 Contributing

Feel free to open issues and pull requests if you have suggestions for improvements!
