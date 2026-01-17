import { Module } from '@nestjs/common';
import { MongooseModule } from '@nestjs/mongoose';
import { ProfileController } from './profile.controller';
import { InternalController } from './internal.controller';
import { ProfileService } from './profile.service';
import { Profile, ProfileSchema } from './schemas/profile.schema';
import { Address, AddressSchema } from './schemas/address.schema';

@Module({
  imports: [
    MongooseModule.forFeature([
      { name: Profile.name, schema: ProfileSchema },
      { name: Address.name, schema: AddressSchema },
    ]),
  ],
  controllers: [ProfileController, InternalController],
  providers: [ProfileService],
  exports: [ProfileService],
})
export class ProfileModule {}
