import { Module } from '@nestjs/common';
import { ProfileModule } from '../profile/profile.module';
import { UserRegisteredListener } from './user-registered.listener';

@Module({
  imports: [ProfileModule],
  providers: [UserRegisteredListener],
})
export class ListenersModule {}
