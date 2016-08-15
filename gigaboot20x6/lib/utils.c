// Copyright 2016 The Fuchsia Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

#include <efi.h>
#include <efilib.h>

#include <utils.h>
#include <printf.h>

const static char *efi_error_labels[] = {
    "EFI_SUCCESS",
    "EFI_LOAD_ERROR",
    "EFI_INVALID_PARAMETER",
    "EFI_UNSUPPORTED",
    "EFI_BAD_BUFFER_SIZE",
    "EFI_BUFFER_TOO_SMALL",
    "EFI_NOT_READY",
    "EFI_DEVICE_ERROR",
    "EFI_WRITE_PROTECTED",
    "EFI_OUT_OF_RESOURCES",
    "EFI_VOLUME_CORRUPTED",
    "EFI_VOLUME_FULL",
    "EFI_NO_MEDIA",
    "EFI_MEDIA_CHANGED",
    "EFI_NOT_FOUND",
    "EFI_ACCESS_DENIED",
    "EFI_NO_RESPONSE",
    "EFI_NO_MAPPING",
    "EFI_TIMEOUT",
    "EFI_NOT_STARTED",
    "EFI_ALREADY_STARTED",
    "EFI_ABORTED",
    "EFI_ICMP_ERROR",
    "EFI_TFTP_ERROR",
    "EFI_PROTOCOL_ERROR",
    "EFI_INCOMPATIBLE_VERSION",
    "EFI_SECURITY_VIOLATION",
    "EFI_CRC_ERROR",
    "EFI_END_OF_MEDIA",
    "EFI_END_OF_FILE",
    "EFI_INVALID_LANGUAGE",
    "EFI_COMPROMISED_DATA",
};

// Useful GUID Constants Not Defined by -lefi
EFI_GUID SimpleFileSystemProtocol = SIMPLE_FILE_SYSTEM_PROTOCOL;
EFI_GUID FileInfoGUID = EFI_FILE_INFO_ID;

// -lefi has its own globals, but this may end up not
// depending on that, so let's not depend on those
EFI_SYSTEM_TABLE* gSys;
EFI_HANDLE gImg;
EFI_BOOT_SERVICES* gBS;
SIMPLE_TEXT_OUTPUT_INTERFACE* gConOut;

void InitGoodies(EFI_HANDLE img, EFI_SYSTEM_TABLE* sys) {
    gSys = sys;
    gImg = img;
    gBS = sys->BootServices;
    gConOut = sys->ConOut;
}

void WaitAnyKey(void) {
    SIMPLE_INPUT_INTERFACE* sii = gSys->ConIn;
    EFI_INPUT_KEY key;
    while (sii->ReadKeyStroke(sii, &key) != EFI_SUCCESS)
        ;
}

void Fatal(const char* msg, EFI_STATUS status) {
    printf("\nERROR: %s (%s)\n", msg, efi_strerror(status));
    WaitAnyKey();
    gBS->Exit(gImg, 1, 0, NULL);
}

CHAR16* HandleToString(EFI_HANDLE h) {
    EFI_DEVICE_PATH* path = DevicePathFromHandle(h);
    if (path == NULL)
        return L"<NoPath>";
    CHAR16* str = DevicePathToStr(path);
    if (str == NULL)
        return L"<NoString>";
    return str;
}

EFI_STATUS OpenProtocol(EFI_HANDLE h, EFI_GUID* guid, void** ifc) {
    return gBS->OpenProtocol(h, guid, ifc, gImg, NULL,
                             EFI_OPEN_PROTOCOL_BY_HANDLE_PROTOCOL);
}

EFI_STATUS CloseProtocol(EFI_HANDLE h, EFI_GUID* guid) {
    return gBS->CloseProtocol(h, guid, gImg, NULL);
}

const char *efi_strerror(EFI_STATUS status)
{
    size_t i = (~EFI_ERROR_MASK & status);
    if (i < sizeof(efi_error_labels)/sizeof(efi_error_labels[0])) {
        return efi_error_labels[i];
    }

    return "<Unknown error>";
}
