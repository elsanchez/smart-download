-- Índices para optimizar queries comunes

-- Downloads: buscar por status (cola de descargas)
CREATE INDEX idx_downloads_status ON downloads(status);

-- Downloads: ordenar por fecha de creación (recientes)
CREATE INDEX idx_downloads_created_at ON downloads(created_at DESC);

-- Downloads: buscar por plataforma
CREATE INDEX idx_downloads_platform ON downloads(platform);

-- Downloads: buscar descargas activas
CREATE INDEX idx_downloads_active ON downloads(status)
WHERE status IN ('downloading', 'processing');

-- Accounts: buscar por plataforma
CREATE INDEX idx_accounts_platform ON accounts(platform);

-- Accounts: obtener cuenta activa rápidamente
CREATE INDEX idx_accounts_active ON accounts(platform, is_active)
WHERE is_active = 1;
