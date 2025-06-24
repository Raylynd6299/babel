# Babel - Polyfy

## Develop
```
# Levantar solo las bases de datos
docker-compose -f docker-compose.dev.yml up -d postgres redis adminer

# Verificar que estén funcionando
docker-compose -f docker-compose.dev.yml ps
```

## Estructrua Proy
├── cmd/                          # Entry points
│   ├── auth-service/
│   ├── content-service/
│   ├── progress-service/
│   ├── vocabulary-service/ 
│   ├── phonetic-service/

│   ├── gamification-service/
│   ├── social-service/
│   ├── analytics-service/
│   ├── notification-service/

│   └── api-gateway/
├── internal/                     # Private application code
│   ├── auth/
│   ├── content/
│   ├── progress/
│   ├── vocabulary/
│   ├── phonetic/
│   ├── gamification/
│   ├── social/
│   ├── analytics/
│   ├── notification/
│   └── shared/
│       ├── config/
│       ├── database/
│       ├── middleware/
│       ├── models/
│       ├── utils/
│       └── validation/
├── pkg/                          # Public library code
│   ├── jwt/
│   ├── logger/
│   ├── cache/
│   ├── http/
│   └── errors/
├── api/                          # API definitions
│   ├── proto/                    # gRPC definitions
│   └── openapi/                  # REST API specs
├── migrations/                   # Database migrations
├── docker/                       # Docker configurations
├── scripts/                      # Build and deployment scripts
├── go.mod
└── go.sum