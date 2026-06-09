import { Test, TestingModule } from '@nestjs/testing';
import { ProfileController } from './controllers/profile.controller';
import { ProfileService } from './services/profile.service';
import { UpdateProfileDto } from './dto/update-profile.dto';
import { CreateAddressDto } from './dto/address.dto';
import { RegisterShopDto } from './dto/register-shop.dto';
import { AddressType } from './schemas/address.schema';

describe('ProfileController', () => {
  let controller: ProfileController;
  let service: ProfileService;

  const mockProfileService = {
    getOrCreateProfile: jest.fn(),
    updateProfile: jest.fn(),
    registerShop: jest.fn(),
    updateShop: jest.fn(),
    getAddresses: jest.fn(),
    addAddress: jest.fn(),
    getAddressById: jest.fn(),
    setDefaultAddress: jest.fn(),
    deleteAddress: jest.fn(),
  };

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      controllers: [ProfileController],
      providers: [
        {
          provide: ProfileService,
          useValue: mockProfileService,
        },
      ],
    }).compile();

    controller = module.get<ProfileController>(ProfileController);
    service = module.get<ProfileService>(ProfileService);

    jest.clearAllMocks();
  });

  it('should be defined', () => {
    expect(controller).toBeDefined();
  });

  describe('getMyProfile', () => {
    it('should return profile with success response', async () => {
      const mockProfile = { userId: 'test-user-id', displayName: 'Test User' };
      mockProfileService.getOrCreateProfile.mockResolvedValue(mockProfile);

      const result = await controller.getMyProfile('test-user-id');

      expect(result).toEqual({
        success: true,
        data: mockProfile,
      });
      expect(mockProfileService.getOrCreateProfile).toHaveBeenCalledWith(
        'test-user-id',
      );
    });
  });

  describe('updateMyProfile', () => {
    it('should update and return profile', async () => {
      const dto: UpdateProfileDto = { displayName: 'Updated Name' };
      const mockProfile = { userId: 'test-user-id', displayName: 'Updated Name' };
      mockProfileService.updateProfile.mockResolvedValue(mockProfile);

      const result = await controller.updateMyProfile('test-user-id', dto);

      expect(result).toEqual({
        success: true,
        data: mockProfile,
        message: 'Profile updated successfully',
      });
    });
  });

  describe('registerShop', () => {
    it('should register shop and return profile', async () => {
      const dto: RegisterShopDto = {
        shopName: 'Test Shop',
        description: 'A test shop',
      };
      const mockProfile = {
        userId: 'test-user-id',
        shopConfig: { shopName: 'Test Shop' },
      };
      mockProfileService.registerShop.mockResolvedValue(mockProfile);

      const result = await controller.registerShop('test-user-id', dto);

      expect(result).toEqual({
        success: true,
        data: mockProfile,
        message: 'Shop registered successfully',
      });
    });
  });

  describe('getMyAddresses', () => {
    it('should return addresses list', async () => {
      const mockAddresses = [
        { userId: 'test-user-id', isDefault: true },
        { userId: 'test-user-id', isDefault: false },
      ];
      mockProfileService.getAddresses.mockResolvedValue(mockAddresses);

      const result = await controller.getMyAddresses('test-user-id');

      expect(result).toEqual({
        success: true,
        data: mockAddresses,
      });
    });
  });

  describe('addAddress', () => {
    it('should add address and return it', async () => {
      const dto: CreateAddressDto = {
        contactName: 'John Doe',
        phone: '0912345678',
        provinceCode: '01',
        districtCode: '001',
        wardCode: '00001',
        streetLine: '123 Main St',
        type: AddressType.HOME,
      };
      const mockAddress = { ...dto, userId: 'test-user-id', isDefault: true };
      mockProfileService.addAddress.mockResolvedValue(mockAddress);

      const result = await controller.addAddress('test-user-id', dto);

      expect(result).toEqual({
        success: true,
        data: mockAddress,
        message: 'Address added successfully',
      });
    });
  });

  describe('setDefaultAddress', () => {
    it('should set address as default', async () => {
      const mockAddress = {
        _id: 'address-id',
        userId: 'test-user-id',
        isDefault: true,
      };
      mockProfileService.setDefaultAddress.mockResolvedValue(mockAddress);

      const result = await controller.setDefaultAddress(
        'test-user-id',
        'address-id',
      );

      expect(result).toEqual({
        success: true,
        data: mockAddress,
        message: 'Default address updated successfully',
      });
    });
  });

  describe('deleteAddress', () => {
    it('should delete address and return success', async () => {
      mockProfileService.deleteAddress.mockResolvedValue(undefined);

      const result = await controller.deleteAddress('test-user-id', 'address-id');

      expect(result).toEqual({
        success: true,
        data: null,
        message: 'Address deleted successfully',
      });
    });
  });
});
