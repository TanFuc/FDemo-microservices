import { Module } from '@nestjs/common';
import { ConfigModule } from '@nestjs/config';

// Config
import { validationSchema } from './common/config/validation.schema';
import { appConfig, databaseConfig, swaggerConfig } from './common/config';

// Database
import { DatabaseModule } from './database';

// Modules
import { HealthModule } from './modules/health';
import { IdentityModule } from './modules/identity';

@Module({
  imports: [
    // Global configuration
    ConfigModule.forRoot({
      isGlobal: true,
      envFilePath: ['.env.local', '.env'],
      load: [appConfig, databaseConfig, swaggerConfig],
      validationSchema,
      validationOptions: {
        allowUnknown: true,
        abortEarly: false,
      },
    }),

    // Database connection
    DatabaseModule,

    // Feature modules
    HealthModule,
    IdentityModule,
  ],
})
export class AppModule {}
