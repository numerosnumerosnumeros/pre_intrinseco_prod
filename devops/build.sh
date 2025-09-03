# build.sh

# Call from root directory: ./devops/build_dev.sh

# Flags: --clean, --prod

set -e

ROOT_DIR=$(pwd)
BUILD_DIR="build"
CLEAN_BUILD=false
PROD_MODE=false

for arg in "$@"; do
    if [ "$arg" == "--clean" ]; then
        CLEAN_BUILD=true
    fi
    if [ "$arg" == "--prod" ]; then
        PROD_MODE=true
    fi
done

if [ "$PROD_MODE" == "true" ]; then
    echo "Building in production mode..."
else
    echo "Building in development mode..."
fi

# Check if .env exists and make a temporary copy (unless --clean is specified)
if [ -f "$BUILD_DIR/.env" ] && [ "$CLEAN_BUILD" == "false" ]; then
    echo "Found existing .env file, creating backup..."
    cp "$BUILD_DIR/.env" ".env.tmp"
fi

echo "Cleaning build directory..."
rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR"

# Restore .env from backup or generate a new one
if [ -f ".env.tmp" ]; then
    printf "\nRestoring .env file from backup...\n"
    cp ".env.tmp" "$BUILD_DIR/.env"
    rm ".env.tmp"
    echo ".env file restored successfully"
else
    printf "\nGenerating .env file in build directory...\n"
    >"$BUILD_DIR/.env"

    printf "\nFetching parameters from AWS SSM...\n"
    PARAMS=(
        "VITE_BASE_URL"
        "VITE_BASE_URL_DEV"
        "VITE_STRIPE_PK"
        "VITE_STRIPE_PK_TEST"
        "VITE_MAX_PERIODS_PER_TICKER"
        "VITE_MAX_TICKERS"
        "VITE_EMAIL_CONTACT"
    )

    for param in "${PARAMS[@]}"; do
        param_name=$(basename "$param")
        param_value=$(aws ssm get-parameter --name "$param" --with-decryption --query "Parameter.Value" --output text)
        echo "$param_name=$param_value" >>"$BUILD_DIR/.env"
    done

    echo "AWS_REGION=eu-central-1" >>"$BUILD_DIR/.env"
    if [ "$PROD_MODE" == "true" ]; then
        echo "DEV_MODE=false" >>"$BUILD_DIR/.env"
    else
        echo "DEV_MODE=true" >>"$BUILD_DIR/.env"
    fi
    echo ".env file generated successfully"
fi

printf "\nBuilding the backend...\n"
cd "$ROOT_DIR/backend"
go mod tidy
if [ "$PROD_MODE" == "true" ]; then
    GOOS=linux GOARCH=arm CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o "$ROOT_DIR/$BUILD_DIR/nodofinance"
else
    go build -race -o "$ROOT_DIR/$BUILD_DIR/nodofinance"
fi
cd "$ROOT_DIR"

printf "\nBuilding the frontend...\n"
cd "$ROOT_DIR/frontend"
PDF_JS_VERSION="5.3.31"
PDFJS_DIR="$ROOT_DIR/frontend/src/lib/pdfjs"

mkdir -p "$PDFJS_DIR"

# Check if PDF.js files already exist (download if they don't or if --clean is specified)
if [ ! -f "$PDFJS_DIR/pdf.mjs" ] || [ "$CLEAN_BUILD" == "true" ]; then
    printf "Downloading PDF.js version ${PDF_JS_VERSION}..."
    curl -s -L "https://github.com/mozilla/pdf.js/releases/download/v${PDF_JS_VERSION}/pdfjs-${PDF_JS_VERSION}-dist.zip" -o pdfjs-dist.zip
    unzip -q -j -o pdfjs-dist.zip "build/pdf.mjs" "build/pdf.worker.mjs" "build/pdf.mjs.map" "build/pdf.worker.mjs.map" -d "$ROOT_DIR/frontend/src/lib/pdfjs/"
    rm pdfjs-dist.zip
    printf "PDF.js ${PDF_JS_VERSION} installed to frontend/src/lib/pdfjs/"
else
    printf "PDF.js files already exist in frontend/src/lib/pdfjs/, skipping download\n"
fi

npm install

if [ "$PROD_MODE" == "true" ]; then
    VITE_USE_DEV=false npm run build
else
    VITE_USE_DEV=true npm run build
fi

# Upload to S3 if in production mode
if [ "$PROD_MODE" == "true" ]; then
    echo "Uploading build to S3..."
    cd "$ROOT_DIR/$BUILD_DIR"
    tar --exclude="./build.tar.gz" --exclude="./.env" -czvf build.tar.gz .
    aws s3 rm s3://compiled-prod-nodofinance/ --recursive
    aws s3 cp build.tar.gz s3://compiled-prod-nodofinance/build.tar.gz
    echo "Build uploaded to S3 successfully."
else
    echo "Skipping S3 upload in development mode."
fi

echo "Build completed successfully."
