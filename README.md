# nex-gen-cms

## How to Run locally?

### Prerequisites:
1. **Install Dbservice:** Install and run it locally following the steps mentioned over [here](https://github.com/avantifellows/db-service/blob/main/docs/INSTALLATION.md).
2. **Install Go:** Make sure you have Go installed (you can check by running `go version`). If it’s not installed, download and install it from [golang.org](https://go.dev/dl/).

### Getting started:
To run the CMS locally, follow these steps:
1. Clone the repository to your local machine.
   
   ```
   git clone https://github.com/avantifellows/nex-gen-cms.git
   ```
2. Create `.env` file at project root directory.
3. Add following 2 key-value pairs in `.env` file.

   ```
   DB_SERVICE_ENDPOINT = http://localhost:4000/api/
   DB_SERVICE_TOKEN = <BEARER_TOKEN used in .env file of your local db-service project>
   ```
4. Navigate to the project directory.
 
   ```
   cd <path to local project root folder>
   ```
5. Run this command to download all necessary dependencies for the project.

   ```
   go mod tidy
   ```
6. Run the application by running:

   ```
   go run cmd/main.go
   ```
7. Open your browser and go to http://localhost:8080 to view the application.
