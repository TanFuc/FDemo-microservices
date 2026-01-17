import { Test, TestingModule } from '@nestjs/testing';
import { getModelToken, getConnectionToken } from '@nestjs/mongoose';
import { Model, Connection, Types } from 'mongoose';
import { NotFoundException, ConflictException } from '@nestjs/common';
import { ProfileService } from './profile.service';
import { Profile, ProfileDocument } from './schemas/profile.schema';
import { Address, AddressDocument } from './schemas/address.schema';
import { CreateAddressDto } from './dto/address.dto';
import { RegisterShopDto } from './dto/register-shop.dto';
import { AddressType } from './schemas/address.schema';

describe('ProfileService', () => {
  let service: ProfileService;
  let profileModel: Model<ProfileDocument>;
  let addressModel: Model<AddressDocument>;
  let connection: Connection;

  const mockProfileModel = {
    findOne: jest.fn(),
    create: jest.fn(),
  };

  const mockAddressModel = {
    find: jest.fn(),
    findOne: jest.fn(),
    findOneAndUpdate: jest.fn(),
    create: jest.fn(),
    countDocuments: jest.fn(),
    updateMany: jest.fn(),
    deleteOne: jest.fn(),
  };

  const mockSession = {
    startTransaction: jest.fn(),
    commitTransaction: jest.fn(),
    abortTransaction: jest.fn(),
    endSession: jest.fn(),
  };

  const mockConnection = {
    startSession: jest.fn().mockResolvedValue(mockSession),
  };

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      providers: [
        ProfileService,
        {
          provide: getModelToken(Profile.name),
          useValue: mockProfileModel,
        },
        {
          provide: getModelToken(Address.name),
          useValue: mockAddressModel,
        },
        {
          provide: getConnectionToken(),
          useValue: mockConnection,
        },
      ],
    }).compile();

    service = module.get<ProfileService>(ProfileService);
    profileModel = module.get<Model<ProfileDocument>>(
      getModelToken(Profile.name),
    );
    addressModel = module.get<Model<AddressDocument>>(
      getModelToken(Address.name),
    );
    connection = module.get<Connection>(getConnectionToken());

    jest.clearAllMocks();
  });

  it('should be defined', () => {
    expect(service).toBeDefined();
  });

  describe('getOrCreateProfile', () => {
    it('should return existing profile if found', async () => {
      const mockProfile = {
        userId: 'test-user-id',
        displayName: 'Test User',
      };

      mockProfileModel.findOne.mockReturnValue({
        exec: jest.fn().mockResolvedValue(mockProfile),
      });

      const result = await service.getOrCreateProfile('test-user-id');

      expect(result).toEqual(mockProfile);
      expect(mockProfileModel.findOne).toHaveBeenCalledWith({
        userId: 'test-user-id',
      });
    });

    it('should create new profile if not found', async () => {
      const mockProfile = {
        userId: 'test-user-id',
        displayName: 'User-r-id',
      };

      mockProfileModel.findOne.mockReturnValue({
        exec: jest.fn().mockResolvedValue(null),
      });
      mockProfileModel.create.mockResolvedValue(mockProfile);

      const result = await service.getOrCreateProfile('test-user-id');

      expect(result).toEqual(mockProfile);
      expect(mockProfileModel.create).toHaveBeenCalledWith({
        userId: 'test-user-id',
        displayName: 'User-r-id',
      });
    });
  });

  describe('addAddress', () => {
    const mockAddressDto: CreateAddressDto = {
      contactName: 'John Doe',
      phone: '0912345678',
      provinceCode: '01',
      districtCode: '001',
      wardCode: '00001',
      streetLine: '123 Main St',
      isDefault: false,
      type: AddressType.HOME,
    };

    it('should set isDefault to true if first address', async () => {
      mockAddressModel.countDocuments.mockResolvedValue(0);
      mockAddressModel.create.mockResolvedValue({
        ...mockAddressDto,
        userId: 'test-user-id',
        isDefault: true,
      });

      const result = await service.addAddress('test-user-id', mockAddressDto);

      expect(result.isDefault).toBe(true);
      expect(mockAddressModel.create).toHaveBeenCalledWith({
        ...mockAddressDto,
        userId: 'test-user-id',
        isDefault: true,
      });
    });

    it('should unset previous default if new address is default', async () => {
      const dtoWithDefault = { ...mockAddressDto, isDefault: true };

      mockAddressModel.countDocuments.mockResolvedValue(1);
      mockAddressModel.updateMany.mockResolvedValue({});
      mockAddressModel.create.mockResolvedValue({
        ...dtoWithDefault,
        userId: 'test-user-id',
      });

      await service.addAddress('test-user-id', dtoWithDefault);

      expect(mockAddressModel.updateMany).toHaveBeenCalledWith(
        { userId: 'test-user-id' },
        { $set: { isDefault: false } },
      );
    });
  });

  describe('setDefaultAddress', () => {
    it('should throw NotFoundException for invalid addressId', async () => {
      await expect(
        service.setDefaultAddress('test-user-id', 'invalid-id'),
      ).rejects.toThrow(NotFoundException);
    });

    it('should set address as default using transaction', async () => {
      const validId = new Types.ObjectId().toString();
      const mockAddress = {
        _id: validId,
        userId: 'test-user-id',
        isDefault: true,
      };

      mockAddressModel.updateMany.mockResolvedValue({});
      mockAddressModel.findOneAndUpdate.mockReturnValue({
        exec: jest.fn().mockResolvedValue(mockAddress),
      });

      const result = await service.setDefaultAddress('test-user-id', validId);

      expect(result).toEqual(mockAddress);
      expect(mockSession.commitTransaction).toHaveBeenCalled();
      expect(mockSession.endSession).toHaveBeenCalled();
    });

    it('should abort transaction if address not found', async () => {
      const validId = new Types.ObjectId().toString();

      mockAddressModel.updateMany.mockResolvedValue({});
      mockAddressModel.findOneAndUpdate.mockReturnValue({
        exec: jest.fn().mockResolvedValue(null),
      });

      await expect(
        service.setDefaultAddress('test-user-id', validId),
      ).rejects.toThrow(NotFoundException);

      expect(mockSession.abortTransaction).toHaveBeenCalled();
      expect(mockSession.endSession).toHaveBeenCalled();
    });
  });

  describe('registerShop', () => {
    const mockShopDto: RegisterShopDto = {
      shopName: 'Test Shop',
      description: 'A test shop',
    };

    it('should throw ConflictException if shop name is taken', async () => {
      mockProfileModel.findOne.mockReturnValue({
        exec: jest.fn().mockResolvedValue({ shopConfig: { shopName: 'Test Shop' } }),
      });

      await expect(
        service.registerShop('test-user-id', mockShopDto),
      ).rejects.toThrow(ConflictException);
    });

    it('should register shop successfully', async () => {
      const mockProfile = {
        userId: 'test-user-id',
        displayName: 'Test User',
        shopConfig: undefined,
        save: jest.fn().mockResolvedValue(true),
      };

      mockProfileModel.findOne
        .mockReturnValueOnce({
          exec: jest.fn().mockResolvedValue(null),
        })
        .mockReturnValueOnce({
          exec: jest.fn().mockResolvedValue(mockProfile),
        });

      const result = await service.registerShop('test-user-id', mockShopDto);

      expect(mockProfile.save).toHaveBeenCalled();
      expect(mockProfile.shopConfig).toEqual({
        shopName: 'Test Shop',
        description: 'A test shop',
        logoUrl: '',
        pickupAddressId: undefined,
      });
    });
  });

  describe('getAddresses', () => {
    it('should return addresses sorted by isDefault', async () => {
      const mockAddresses = [
        { userId: 'test-user-id', isDefault: true },
        { userId: 'test-user-id', isDefault: false },
      ];

      mockAddressModel.find.mockReturnValue({
        sort: jest.fn().mockReturnValue({
          exec: jest.fn().mockResolvedValue(mockAddresses),
        }),
      });

      const result = await service.getAddresses('test-user-id');

      expect(result).toEqual(mockAddresses);
      expect(mockAddressModel.find).toHaveBeenCalledWith({
        userId: 'test-user-id',
      });
    });
  });

  describe('getAddressById', () => {
    it('should throw NotFoundException for invalid addressId', async () => {
      await expect(
        service.getAddressById('test-user-id', 'invalid-id'),
      ).rejects.toThrow(NotFoundException);
    });

    it('should return address if found', async () => {
      const validId = new Types.ObjectId().toString();
      const mockAddress = { _id: validId, userId: 'test-user-id' };

      mockAddressModel.findOne.mockReturnValue({
        exec: jest.fn().mockResolvedValue(mockAddress),
      });

      const result = await service.getAddressById('test-user-id', validId);

      expect(result).toEqual(mockAddress);
    });

    it('should throw NotFoundException if address not found', async () => {
      const validId = new Types.ObjectId().toString();

      mockAddressModel.findOne.mockReturnValue({
        exec: jest.fn().mockResolvedValue(null),
      });

      await expect(
        service.getAddressById('test-user-id', validId),
      ).rejects.toThrow(NotFoundException);
    });
  });

  describe('deleteAddress', () => {
    it('should throw NotFoundException for invalid addressId', async () => {
      await expect(
        service.deleteAddress('test-user-id', 'invalid-id'),
      ).rejects.toThrow(NotFoundException);
    });

    it('should delete address and set next as default if was default', async () => {
      const validId = new Types.ObjectId().toString();
      const mockAddress = {
        _id: validId,
        userId: 'test-user-id',
        isDefault: true,
      };
      const nextAddress = {
        _id: new Types.ObjectId().toString(),
        userId: 'test-user-id',
        isDefault: false,
        save: jest.fn().mockResolvedValue(true),
      };

      mockAddressModel.findOne
        .mockReturnValueOnce({
          exec: jest.fn().mockResolvedValue(mockAddress),
        })
        .mockReturnValueOnce({
          sort: jest.fn().mockReturnValue({
            exec: jest.fn().mockResolvedValue(nextAddress),
          }),
        });
      mockAddressModel.deleteOne.mockResolvedValue({});

      await service.deleteAddress('test-user-id', validId);

      expect(mockAddressModel.deleteOne).toHaveBeenCalled();
      expect(nextAddress.save).toHaveBeenCalled();
      expect(nextAddress.isDefault).toBe(true);
    });
  });
});
