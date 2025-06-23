# Babel - Polyfy

## Infrastructure

┌─────────────────────────────────────────────────────────────┐
│                   API GATEWAY (Go)                          │
├─────────────────────────────────────────────────────────────┤
│  JWT Auth │ Rate Limiting │ CORS │ Request Logging          │
│  Middleware Chain │ Request Validation │ Response Caching   │
└─────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────┐
│                 GO MICROSERVICES                            │
├─────────────────────────────────────────────────────────────┤
│ Auth Service │ Content Service │ Progress Service │ Vocab   │
│ User Service │ Analytics Service │ Social Service │ Game    │
│ Notification │ Recommendation │ Phonetic Service │ Export  │
└─────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────┐
│                    DATA LAYER                               │
├─────────────────────────────────────────────────────────────┤
│  PostgreSQL  │  Redis Cache  │  File Storage  │  InfluxDB   │
└─────────────────────────────────────────────────────────────┘

## Develop
```
# Levantar solo las bases de datos
docker-compose -f docker-compose.dev.yml up -d postgres redis adminer

# Verificar que estén funcionando
docker-compose -f docker-compose.dev.yml ps
```