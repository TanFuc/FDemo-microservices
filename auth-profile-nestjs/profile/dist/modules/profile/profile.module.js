"use strict";
var __decorate = (this && this.__decorate) || function (decorators, target, key, desc) {
    var c = arguments.length, r = c < 3 ? target : desc === null ? desc = Object.getOwnPropertyDescriptor(target, key) : desc, d;
    if (typeof Reflect === "object" && typeof Reflect.decorate === "function") r = Reflect.decorate(decorators, target, key, desc);
    else for (var i = decorators.length - 1; i >= 0; i--) if (d = decorators[i]) r = (c < 3 ? d(r) : c > 3 ? d(target, key, r) : d(target, key)) || r;
    return c > 3 && r && Object.defineProperty(target, key, r), r;
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.ProfileModule = void 0;
const common_1 = require("@nestjs/common");
const microservices_1 = require("@nestjs/microservices");
const path_1 = require("path");
const profile_controller_1 = require("./controllers/profile.controller");
const internal_controller_1 = require("./controllers/internal.controller");
const profile_service_1 = require("./services/profile.service");
let ProfileModule = class ProfileModule {
};
exports.ProfileModule = ProfileModule;
exports.ProfileModule = ProfileModule = __decorate([
    (0, common_1.Module)({
        imports: [
            microservices_1.ClientsModule.register([
                {
                    name: 'AUTH_PACKAGE',
                    transport: microservices_1.Transport.GRPC,
                    options: {
                        package: 'auth',
                        protoPath: (0, path_1.join)(__dirname, '../../protos/auth.proto'),
                        url: process.env.AUTH_GRPC_URL || '0.0.0.0:50051',
                    },
                },
            ]),
        ],
        controllers: [profile_controller_1.ProfileController, internal_controller_1.InternalController],
        providers: [profile_service_1.ProfileService],
        exports: [profile_service_1.ProfileService],
    })
], ProfileModule);
//# sourceMappingURL=profile.module.js.map