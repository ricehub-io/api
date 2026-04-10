# Contributing to the API

First off, thank you for considering contributing <3.
All types of contributions are encouraged and valued.

## How to contribute

- Fork the repository
- Create a new branch with clear and concise name (e.g. `feature/feature-name` or `fix/bug-name`)
- Make your changes
- Open a pull request and don't forget to provide a detailed description of changes you've made

## Coding conventions

### Function naming by layer

The project has three layers, each with its own naming convention:

**Repository** (`internal/repository`) - maps to database operations:

| Action           | Verb     | Example               |
| ---------------- | -------- | --------------------- |
| Insert a row     | `Insert` | `InsertComment`       |
| Select one row   | `Find`   | `FindCommentByID`     |
| Select many rows | `Fetch`  | `FetchRecentComments` |
| Update a row     | `Update` | `UpdateComment`       |
| Delete a row     | `Delete` | `DeleteComment`       |

**Service** (`internal/services`) - maps to business/domain operations:

| Action             | Verb     | Example          |
| ------------------ | -------- | ---------------- |
| Create a resource  | `Create` | `CreateComment`  |
| Get one resource   | `Get`    | `GetCommentByID` |
| Get many resources | `List`   | `ListComments`   |
| Update a resource  | `Update` | `UpdateComment`  |
| Delete a resource  | `Delete` | `DeleteComment`  |

Use a descriptive domain verb instead when the operation isn't plain CRUD (e.g. `PublishRice`, `BanUser`).

**Handlers** (`internal/handlers`) - maps to HTTP endpoints, uses the same verbs as services:

| HTTP method  | Verb     | Example          |
| ------------ | -------- | ---------------- |
| POST         | `Create` | `CreateComment`  |
| GET (single) | `Get`    | `GetCommentByID` |
| GET (list)   | `List`   | `ListComments`   |
| PUT / PATCH  | `Update` | `UpdateComment`  |
| DELETE       | `Delete` | `DeleteComment`  |

### Service layer rules

Services contain all business logic. Keep them free of HTTP concerns.

**Do:**

- Validate business rules (e.g. blacklist checks, ownership checks)
- Coordinate multiple repository calls
- Hash passwords, generate tokens, call external integrations (Polar, gRPC)
- Return domain models (`models.User`, `models.Rice`, etc.)
- Return predefined errors from the `errs` package
- Use a result struct when returning more than two values

**Don't:**

- Import `github.com/gin-gonic/gin` or `net/http`
- Return response DTOs (types with `json` tags meant for HTTP responses), instead return domain models and let handlers call `.ToDTO()`
- Call the repository directly from a handler, all business logic goes through the service layer

## Before submitting

- Make sure the project builds
- Check if newly added features work as intended
- Keep coding conventions consistent with the existing code

## How to report a bug

If you find a bug, please open a GitHub Issue and include:

- What exactly happened
- Steps to reproduce (if known)
- Expected behavior
- Screenshots (if applicable)

## What can I work on?

You can find a list of things to work on in the **Issues** tab.
Each issue has a corresponding label attached to it.

Some of the label meanings:

- **help wanted:** The task has not been claimed by anyone. Feel free to start working on it, but please leave a comment that you're working on it.
- **good first issue:** Beginner-friendly tasks for people who are beginning their journey with contributing or programming.
- **enhancement:** Improvements that are not bugs. For example UI/UX updates, performance improvements, code refactoring, or new features.
- **question:** The issue is a discussion or question about the project rather than a task.

Other than that, you are welcome to improve anything in the project.

If you believe something can be implemented better, feel free to open an issue and explain your idea before submitting a PR.

Major architectural changes should be discussed first.

## Contributing rices (content)

The biggest way to support this project right now is by sharing your own rice on the website.

If you use Linux or any Unix-like system, consider submitting your setup.

More content makes the platform more useful and inspiring for others.

## Feature requests

To request a feature open a new GitHub issue with details of how you want it to work, where should it be implemented, and other informations that might be useful.

---

If you're unsure about anything, open an issue and ask first.

You can also join the Discord server where discussion is often quicker:
https://discord.gg/z7Zu8MeTdG
