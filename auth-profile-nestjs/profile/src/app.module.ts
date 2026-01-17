import { Module } from '@nestjs/common';
import { MongooseModule } from '@nestjs/mongoose';
import { ProfileModule } from './modules/profile/profile.module';
import { ListenersModule } from './modules/listeners/listeners.module';

@Module({
  imports: [
    MongooseModule.forRoot(
      process.env.MONGODB_URI || 'mongodb://localhost:27017/profile-service',
    ),
    ProfileModule,
    ListenersModule,
  ],
})
export class AppModule {}
