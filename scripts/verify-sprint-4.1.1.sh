#!/bin/bash
# Sprint 4.1.1 验收脚本

set -e

echo "=========================================="
echo "Sprint 4.1.1 Contract Alignment Verification"
echo "=========================================="
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

check() {
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓${NC} $1"
    else
        echo -e "${RED}✗${NC} $1"
        exit 1
    fi
}

# 1. Go Module Check
echo "1. Checking Go module path..."
cd server
if grep -q "github.com/zhouwu97/key-cabinet/server" go.mod; then
    check "Go module path updated"
else
    echo -e "${RED}✗${NC} Go module path not updated"
    exit 1
fi

# 2. Go Build Check
echo ""
echo "2. Building Go backend..."
go build ./cmd/api
check "Go build successful"
rm -f api api.exe

# 3. Migration Files Check
echo ""
echo "3. Checking migration files..."
EXPECTED_MIGRATIONS=(
    "000001_init.up.sql"
    "000001_init.down.sql"
    "000002_reservation_constraint.up.sql"
    "000002_reservation_constraint.down.sql"
    "000003_fix_schema_alignment.up.sql"
    "000003_fix_schema_alignment.down.sql"
    "000004_create_operation_events.up.sql"
    "000004_create_operation_events.down.sql"
    "000005_fix_reservation_constraint.up.sql"
    "000005_fix_reservation_constraint.down.sql"
)

for migration in "${EXPECTED_MIGRATIONS[@]}"; do
    if [ -f "migrations/$migration" ]; then
        echo -e "${GREEN}✓${NC} $migration exists"
    else
        echo -e "${RED}✗${NC} $migration missing"
        exit 1
    fi
done

# 4. Frontend TypeScript Check
echo ""
echo "4. Checking frontend TypeScript..."
cd ../miniprogram
if command -v npm &> /dev/null; then
    npm run check
    check "TypeScript check passed"
else
    echo -e "${YELLOW}⚠${NC} npm not found, skipping frontend check"
fi

# 5. Contract Consistency Check
echo ""
echo "5. Checking contract consistency..."
cd ..

# Check for studentId in TypeScript models (should not exist)
if grep -r "studentId" miniprogram/models/ --include="*.ts" 2>/dev/null | grep -v "studentNo"; then
    echo -e "${RED}✗${NC} Found studentId in frontend models"
    exit 1
else
    check "Frontend models use studentNo"
fi

# Check for studentId in API docs (should not exist)
if grep "studentId" docs/04-API-CONTRACT.md 2>/dev/null; then
    echo -e "${RED}✗${NC} Found studentId in API Contract"
    exit 1
else
    check "API Contract uses studentNo"
fi

# Check for studentId in Data Model docs (should not exist)
if grep "studentId" docs/02-DATA-MODEL.md 2>/dev/null; then
    echo -e "${RED}✗${NC} Found studentId in Data Model"
    exit 1
else
    check "Data Model uses studentNo"
fi

# 6. Check critical files exist
echo ""
echo "6. Checking new infrastructure files..."
FILES_TO_CHECK=(
    "miniprogram/api/http-client.ts"
    "miniprogram/config/index.ts"
    "miniprogram/services/auth/auth-service.ts"
    "miniprogram/services/user/api-user-service.ts"
    ".github/workflows/ci.yml"
    "docs/sprints/SPRINT-4.1.1-CONTRACT-ALIGNMENT.md"
)

for file in "${FILES_TO_CHECK[@]}"; do
    if [ -f "$file" ]; then
        echo -e "${GREEN}✓${NC} $file exists"
    else
        echo -e "${RED}✗${NC} $file missing"
        exit 1
    fi
done

echo ""
echo "=========================================="
echo -e "${GREEN}✓ Sprint 4.1.1 verification passed!${NC}"
echo "=========================================="
echo ""
echo "Next steps:"
echo "1. Apply migrations to database: make migrate-up"
echo "2. Verify database schema: psql ... -c '\d users'"
echo "3. Run CI: git push (GitHub Actions will run)"
echo "4. Start Sprint 4.2: WeChat login implementation"
