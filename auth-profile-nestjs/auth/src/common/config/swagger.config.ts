import { registerAs } from '@nestjs/config';

export const swaggerConfig = registerAs('swagger', () => ({
  enabled: process.env.SWAGGER_ENABLED === 'true',
  title: process.env.SWAGGER_TITLE || 'Tafu Auth API',
  description: process.env.SWAGGER_DESCRIPTION || 'Identity Service API',
  version: process.env.SWAGGER_VERSION || '1.0',
}));
