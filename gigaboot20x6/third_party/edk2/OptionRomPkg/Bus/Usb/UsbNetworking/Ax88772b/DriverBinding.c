 /** @file
  Implement the driver binding protocol for Asix AX88772 Ethernet driver.

  Copyright (c) 2011-2013, Intel Corporation
  All rights reserved. This program and the accompanying materials
  are licensed and made available under the terms and conditions of the BSD License
  which accompanies this distribution.  The full text of the license may be found at
  http://opensource.org/licenses/bsd-license.php

  THE PROGRAM IS DISTRIBUTED UNDER THE BSD LICENSE ON AN "AS IS" BASIS,
  WITHOUT WARRANTIES OR REPRESENTATIONS OF ANY KIND, EITHER EXPRESS OR IMPLIED.

**/

#include "Ax88772.h"

/*
 * CallerID guid appears to be an EDK2 standard to store a pointer to the Nic that
 * the driver is working with. Replicating it here should cover its usage since
 * the uniqueness of the GUID is all that seems to matter.
 */
#define EFI_STANDARD_CALLER_ID_GUID \
  {0xC9DCF469, 0xA7C4, 0x11D5, {0x87, 0xDA, 0x00, 0x06, 0x29, 0x45, 0xC3, 0xB9}}
EFI_GUID CallerIdProtocol = EFI_STANDARD_CALLER_ID_GUID;

EFI_GUID UsbIoProtocol = EFI_USB_IO_PROTOCOL_GUID;
ASIX_DONGLE ASIX_DONGLES[] = {
  { 0x05AC, 0x1402, FLAG_TYPE_AX88772 }, // Apple USB Ethernet Adapter
  // ASIX 88772B
  { 0x0B95, 0x772B, FLAG_TYPE_AX88772B | FLAG_EEPROM_MAC },
  { 0x0000, 0x0000, FLAG_NONE }   // END - Do not remove
};

/**
  Verify the controller type

  @param [in] pThis                Protocol instance pointer.
  @param [in] Controller           Handle of device to test.
  @param [in] pRemainingDevicePath Not used.

  @retval EFI_SUCCESS          This driver supports this device.
  @retval other                This driver does not support this device.

**/
EFI_STATUS
EFIAPI
DriverSupported (
  IN EFI_DRIVER_BINDING * pThis,
  IN EFI_HANDLE Controller,
  IN EFI_DEVICE_PATH * pRemainingDevicePath
  )
{
  EFI_USB_DEVICE_DESCRIPTOR Device;
  EFI_USB_IO_PROTOCOL * pUsbIo;
  EFI_STATUS Status;
  UINT32 Index;

  //
  //  Connect to the USB stack
  //
  Status = gBS->OpenProtocol (
                  Controller,
                  &UsbIoProtocol,
                  (VOID **) &pUsbIo,
                  pThis->DriverBindingHandle,
                  Controller,
                  EFI_OPEN_PROTOCOL_BY_DRIVER
                  );
  if (!EFI_ERROR ( Status )) {

    //
    //  Get the interface descriptor to check the USB class and find a transport
    //  protocol handler.
    //
    Status = pUsbIo->UsbGetDeviceDescriptor ( pUsbIo, &Device );
    if (EFI_ERROR ( Status )) {
    	Status = EFI_UNSUPPORTED;
    }
    else {
      //
      //  Validate the adapter
      //
      for (Index = 0; ASIX_DONGLES[Index].VendorId != 0; Index++) {
        if (ASIX_DONGLES[Index].VendorId == Device.IdVendor &&
            ASIX_DONGLES[Index].ProductId == Device.IdProduct) {
              break;
        }
      }

      if (ASIX_DONGLES[Index].VendorId == 0)
         Status = EFI_UNSUPPORTED;
    }

    //
    //  Done with the USB stack
    //
    gBS->CloseProtocol (
           Controller,
           &UsbIoProtocol,
           pThis->DriverBindingHandle,
           Controller
           );
  }
  return Status;
}


/**
  Start this driver on Controller by opening UsbIo and DevicePath protocols.
  Initialize PXE structures, create a copy of the Controller Device Path with the
  NIC's MAC address appended to it, install the NetworkInterfaceIdentifier protocol
  on the newly created Device Path.

  @param [in] pThis                Protocol instance pointer.
  @param [in] Controller           Handle of device to work with.
  @param [in] pRemainingDevicePath Not used, always produce all possible children.

  @retval EFI_SUCCESS          This driver is added to Controller.
  @retval other                This driver does not support this device.

**/
EFI_STATUS
EFIAPI
DriverStart (
  IN EFI_DRIVER_BINDING * pThis,
  IN EFI_HANDLE Controller,
  IN EFI_DEVICE_PATH * pRemainingDevicePath
  )
{

    EFI_STATUS						Status;
    NIC_DEVICE						*pNicDevice;
    UINTN							LengthInBytes;
    EFI_DEVICE_PATH                 *ParentDevicePath = NULL;
    MAC_ADDR_DEVICE_PATH            MacDeviceNode;
    EFI_USB_DEVICE_DESCRIPTOR       Device;
    UINT32                          Index;

    //
	//  Allocate the device structure
	//
	LengthInBytes = sizeof ( *pNicDevice );
	Status = gBS->AllocatePool (
                  EfiRuntimeServicesData,
                  LengthInBytes,
                  (VOID **) &pNicDevice
                  );

	if (EFI_ERROR (Status)) {
		DEBUG (D_ERROR, L"gBS->AllocatePool:pNicDevice ERROR Status = %r\n", Status);
		goto EXIT;
	}

  //
  //  Set the structure signature
  //
  ZeroMem ( pNicDevice, LengthInBytes );
  pNicDevice->Signature = DEV_SIGNATURE;

	Status = gBS->OpenProtocol (
                    Controller,
                    &UsbIoProtocol,
                    (VOID **) &pNicDevice->pUsbIo,
                    pThis->DriverBindingHandle,
                    Controller,
                    EFI_OPEN_PROTOCOL_BY_DRIVER
                    );

	if (EFI_ERROR (Status)) {
		DEBUG (D_ERROR, L"gBS->OpenProtocol:EFI_USB_IO_PROTOCOL ERROR Status = %r\n", Status);
		gBS->FreePool ( pNicDevice );
		goto EXIT;
	}

	//
  //  Initialize the simple network protocol
  //
	Status = SN_Setup ( pNicDevice );

	if (EFI_ERROR(Status)){
	   DEBUG (D_ERROR, L"SN_Setup ERROR Status = %r\n", Status);
	   gBS->CloseProtocol (
					Controller,
					&UsbIoProtocol,
					pThis->DriverBindingHandle,
					Controller
					);
		  gBS->FreePool ( pNicDevice );
		  goto EXIT;
  }

  Status = pNicDevice->pUsbIo->UsbGetDeviceDescriptor ( pNicDevice->pUsbIo, &Device );
  if (EFI_ERROR ( Status )) {
     gBS->CloseProtocol (
               Controller,
               &UsbIoProtocol,
               pThis->DriverBindingHandle,
               Controller
               );
     gBS->FreePool ( pNicDevice );
              goto EXIT;
  } else {
      //
      //  Validate the adapter
      //
      for (Index = 0; ASIX_DONGLES[Index].VendorId != 0; Index++) {
          if (ASIX_DONGLES[Index].VendorId == Device.IdVendor &&
              ASIX_DONGLES[Index].ProductId == Device.IdProduct) {
                break;
          }
      }

      if (ASIX_DONGLES[Index].VendorId == 0) {
         gBS->CloseProtocol (
                   Controller,
                   &UsbIoProtocol,
                   pThis->DriverBindingHandle,
                   Controller
                   );
          gBS->FreePool ( pNicDevice );
                   goto EXIT;
      }

      pNicDevice->Flags = ASIX_DONGLES[Index].Flags;
  }

	//
  // Set Device Path
  //
  Status = gBS->OpenProtocol (
                  Controller,
                  &DevicePathProtocol,
                  (VOID **) &ParentDevicePath,
				          pThis->DriverBindingHandle,
                  Controller,
                  EFI_OPEN_PROTOCOL_BY_DRIVER
                  );
	if (EFI_ERROR(Status)) {
        DEBUG (D_ERROR, L"gBS->OpenProtocol:EFI_DEVICE_PATH error. Status = %r\n", Status);
		    gBS->CloseProtocol (
					Controller,
					&UsbIoProtocol,
					pThis->DriverBindingHandle,
					Controller
					);
		  gBS->FreePool ( pNicDevice );
		  goto EXIT;
	}

  ZeroMem (&MacDeviceNode, sizeof (MAC_ADDR_DEVICE_PATH));
  MacDeviceNode.Header.Type = MESSAGING_DEVICE_PATH;
  MacDeviceNode.Header.SubType = MSG_MAC_ADDR_DP;

  SetDevicePathNodeLength (&MacDeviceNode.Header, sizeof (MAC_ADDR_DEVICE_PATH));

  CopyMem (&MacDeviceNode.MacAddress,
      								&pNicDevice->SimpleNetworkData.CurrentAddress,
      								PXE_HWADDR_LEN_ETHER);

  MacDeviceNode.IfType = pNicDevice->SimpleNetworkData.IfType;

  pNicDevice->MyDevPath = AppendDevicePathNode (
                                          ParentDevicePath,
                                          (EFI_DEVICE_PATH *) &MacDeviceNode
                                          );

	pNicDevice->Controller = NULL;

	//
  //  Install both the simple network and device path protocols.
  //
  Status = gBS->InstallMultipleProtocolInterfaces (
                          &pNicDevice->Controller,
                          &CallerIdProtocol,
                          pNicDevice,
                          &SimpleNetworkProtocol,
                          &pNicDevice->SimpleNetwork,
						              &DevicePathProtocol,
						              pNicDevice->MyDevPath,
                          NULL
                          );

	if (EFI_ERROR(Status)){
		DEBUG (D_ERROR, L"gBS->InstallMultipleProtocolInterfaces error. Status = %r\n", Status);
		gBS->CloseProtocol (
					               Controller,
					               &DevicePathProtocol,
					               pThis->DriverBindingHandle,
					               Controller);
	   gBS->CloseProtocol (
					Controller,
					&UsbIoProtocol,
					pThis->DriverBindingHandle,
					Controller
					);
		  gBS->FreePool ( pNicDevice );
		  goto EXIT;
	}

	//
	// Open For Child Device
	//
	Status = gBS->OpenProtocol (
                  Controller,
                  &UsbIoProtocol,
                  (VOID **) &pNicDevice->pUsbIo,
                  pThis->DriverBindingHandle,
                  pNicDevice->Controller,
                  EFI_OPEN_PROTOCOL_BY_CHILD_CONTROLLER
                  );

	if (EFI_ERROR(Status)){
	   gBS->UninstallMultipleProtocolInterfaces (
              &pNicDevice->Controller,
                          &CallerIdProtocol,
                          pNicDevice,
                          &SimpleNetworkProtocol,
                          &pNicDevice->SimpleNetwork,
						              &DevicePathProtocol,
						              pNicDevice->MyDevPath,
                          NULL
                          );
		gBS->CloseProtocol (
					               Controller,
					               &DevicePathProtocol,
					               pThis->DriverBindingHandle,
					               Controller);
	   gBS->CloseProtocol (
					Controller,
					&UsbIoProtocol,
					pThis->DriverBindingHandle,
					Controller
					);
		  gBS->FreePool ( pNicDevice );
	}

EXIT:
	return Status;

}

/**
  Stop this driver on Controller by removing NetworkInterfaceIdentifier protocol and
  closing the DevicePath and PciIo protocols on Controller.

  @param [in] pThis                Protocol instance pointer.
  @param [in] Controller           Handle of device to stop driver on.
  @param [in] NumberOfChildren     How many children need to be stopped.
  @param [in] pChildHandleBuffer   Not used.

  @retval EFI_SUCCESS          This driver is removed Controller.
  @retval EFI_DEVICE_ERROR     The device could not be stopped due to a device error.
  @retval other                This driver was not removed from this device.

**/
EFI_STATUS
EFIAPI
DriverStop (
  IN  EFI_DRIVER_BINDING * pThis,
  IN  EFI_HANDLE Controller,
  IN  UINTN NumberOfChildren,
  IN  EFI_HANDLE * ChildHandleBuffer
  )
{
		BOOLEAN                                   AllChildrenStopped;
		UINTN                                     Index;
		EFI_SIMPLE_NETWORK				  *SimpleNetwork;
		EFI_STATUS                                Status = EFI_SUCCESS;
		NIC_DEVICE								  *pNicDevice;


		//
		// Complete all outstanding transactions to Controller.
		// Don't allow any new transaction to Controller to be started.
		//
		if (NumberOfChildren == 0) {

		  Status = gBS->OpenProtocol (
				                Controller,
				                &SimpleNetworkProtocol,
				                (VOID **) &SimpleNetwork,
				                pThis->DriverBindingHandle,
				                Controller,
				                EFI_OPEN_PROTOCOL_GET_PROTOCOL
				                );

			if (EFI_ERROR(Status)) {
        //
        // This is a 2nd type handle(multi-lun root), it needs to close devicepath
        // and usbio protocol.
        //
        gBS->CloseProtocol (
            Controller,
            &DevicePathProtocol,
            pThis->DriverBindingHandle,
            Controller
            );
        gBS->CloseProtocol (
            Controller,
            &UsbIoProtocol,
            pThis->DriverBindingHandle,
            Controller
            );
        return EFI_SUCCESS;
      }

      pNicDevice = DEV_FROM_SIMPLE_NETWORK ( SimpleNetwork );

      Status = gBS->UninstallMultipleProtocolInterfaces (
				                  Controller,
				                  &CallerIdProtocol,
                          pNicDevice,
                          &SimpleNetworkProtocol,
                          &pNicDevice->SimpleNetwork,
						              &DevicePathProtocol,
						              pNicDevice->MyDevPath,
                          NULL
                          );

      if (EFI_ERROR (Status)) {
        return Status;
      }
		  //
		  // Close the bus driver
		  //
		  Status = gBS->CloseProtocol (
		                  Controller,
		                  &DevicePathProtocol,
		                  pThis->DriverBindingHandle,
		                  Controller
		                  );

		  if (EFI_ERROR(Status)) {
              DEBUG (D_ERROR, L"driver stop: gBS->CloseProtocol:EfiDevicePathProtocol error. Status %r\n", Status);
		  }

		  Status = gBS->CloseProtocol (
		                  Controller,
		                  &UsbIoProtocol,
		                  pThis->DriverBindingHandle,
		                  Controller
		                  );

		  if (EFI_ERROR(Status)) {
              DEBUG (D_ERROR, L"driver stop: gBS->CloseProtocol:EfiUsbIoProtocol error. Status %r\n", Status);
		  }
          return EFI_SUCCESS;
		}

		AllChildrenStopped = TRUE;
		for (Index = 0; Index < NumberOfChildren; Index++) {
				Status = gBS->OpenProtocol (
				                ChildHandleBuffer[Index],
				                &SimpleNetworkProtocol,
				                (VOID **) &SimpleNetwork,
				                pThis->DriverBindingHandle,
				                Controller,
				                EFI_OPEN_PROTOCOL_GET_PROTOCOL
				                );

				if (EFI_ERROR (Status)) {
          AllChildrenStopped = FALSE;
          DEBUG (D_ERROR, L"Fail to stop No.%d multi-lun child handle when opening SimpleNetwork\n", (UINT32)Index);
          continue;
        }

        pNicDevice = DEV_FROM_SIMPLE_NETWORK ( SimpleNetwork );

        gBS->CloseProtocol (
				                    Controller,
				                    &UsbIoProtocol,
				                    pThis->DriverBindingHandle,
				                    ChildHandleBuffer[Index]
				                    );

				Status = gBS->UninstallMultipleProtocolInterfaces (
				                  ChildHandleBuffer[Index],
				                  &CallerIdProtocol,
                          pNicDevice,
                          &SimpleNetworkProtocol,
                          &pNicDevice->SimpleNetwork,
						              &DevicePathProtocol,
						              pNicDevice->MyDevPath,
                          NULL
                          );

        if (EFI_ERROR (Status)) {
            Status = gBS->OpenProtocol (
                  Controller,
                  &UsbIoProtocol,
                  (VOID **) &pNicDevice->pUsbIo,
                  pThis->DriverBindingHandle,
                  ChildHandleBuffer[Index],
                  EFI_OPEN_PROTOCOL_BY_CHILD_CONTROLLER
                  );
        }
        else {
            int i;
            RX_PKT * pCurr = pNicDevice->QueueHead;
            RX_PKT * pFree;

            for ( i = 0 ; i < MAX_QUEUE_SIZE ; i++) {
                 if ( NULL != pCurr ) {
                    pFree = pCurr;
                    pCurr = pCurr->pNext;
                    gBS->FreePool (pFree);
                 }
            }

            if ( NULL != pNicDevice->pRxTest)
						    gBS->FreePool (pNicDevice->pRxTest);

					 if ( NULL != pNicDevice->pTxTest)
						    gBS->FreePool (pNicDevice->pTxTest);

           if ( NULL != pNicDevice->MyDevPath)
					       gBS->FreePool (pNicDevice->MyDevPath);

				    if ( NULL != pNicDevice)
                  gBS->FreePool (pNicDevice);
        }
    }

        if (!AllChildrenStopped) {
                return EFI_DEVICE_ERROR;
        }
        return EFI_SUCCESS;
}


/**
  Driver binding protocol declaration
**/
EFI_DRIVER_BINDING  gDriverBinding = {
  DriverSupported,
  DriverStart,
  DriverStop,
  0xa,
  NULL,
  NULL
};


/**
  Ax88772 driver unload routine.

  @param [in] ImageHandle       Handle for the image.

  @retval EFI_SUCCESS           Image may be unloaded

**/
EFI_STATUS
EFIAPI
DriverUnload (
  IN EFI_HANDLE ImageHandle
  )
{
  UINTN BufferSize;
  UINTN Index;
  UINTN Max;
  EFI_HANDLE * pHandle;
  EFI_STATUS Status;

  //
  //  Determine which devices are using this driver
  //
  BufferSize = 0;
  pHandle = NULL;
  Status = gBS->LocateHandle (
                  ByProtocol,
                  &CallerIdProtocol,
                  NULL,
                  &BufferSize,
                  NULL );
  if ( EFI_BUFFER_TOO_SMALL == Status ) {
    for ( ; ; ) {
      //
      //  One or more block IO devices are present
      //
      Status = gBS->AllocatePool (
                      EfiRuntimeServicesData,
                      BufferSize,
                      (VOID **) &pHandle
                      );
      if ( EFI_ERROR ( Status )) {
        DEBUG (D_ERROR, L"Insufficient memory, failed handle buffer allocation\r\n");
        break;
      }

      //
      //  Locate the block IO devices
      //
      Status = gBS->LocateHandle (
                      ByProtocol,
                      &CallerIdProtocol,
                      NULL,
                      &BufferSize,
                      pHandle );
      if ( EFI_ERROR ( Status )) {
        //
        //  Error getting handles
        //
        break;
      }

      //
      //  Remove any use of the driver
      //
      Max = BufferSize / sizeof ( pHandle[ 0 ]);
      for ( Index = 0; Max > Index; Index++ ) {
        Status = DriverStop ( &gDriverBinding,
                              pHandle[ Index ],
                              0,
                              NULL );
        if ( EFI_ERROR ( Status )) {
          DEBUG (D_ERROR, L"WARNING - Failed to shutdown the driver on handle %08x\r\n", pHandle[Index]);
          break;
        }
      }
      break;
    }
  }
  else {
    if ( EFI_NOT_FOUND == Status ) {
      //
      //  No devices were found
      //
      Status = EFI_SUCCESS;
    }
  }

  //
  //  Free the handle array
  //
  if ( NULL != pHandle ) {
    gBS->FreePool ( pHandle );
  }

  //
  //  Remove the protocols installed by the EntryPoint routine.
  //
  if ( !EFI_ERROR ( Status )) {
    gBS->UninstallMultipleProtocolInterfaces (
            ImageHandle,
            &DriverBindingProtocol,
            &gDriverBinding,
            &ComponentNameProtocol,
            &gComponentName,
            &ComponentName2Protocol,
            &gComponentName2,
            NULL
            );

    DEBUG (D_INIT, L"Removed: ComponentName2Protocol from 0x%08x\r\n", ImageHandle);
    DEBUG (D_INIT, L"Removed: ComponentNameProtocol from 0x%08x\r\n", ImageHandle);
    DEBUG (D_INIT, L"Removed: DriverBindingProtocol from 0x%08x\r\n", ImageHandle);

  }

  return Status;
}

EFI_STATUS
EFIAPI
ax88772_install (
  IN EFI_HANDLE ImageHandle,
  IN EFI_SYSTEM_TABLE * pSystemTable
  )
{
  EFI_STATUS Status;
  EFI_HANDLE *handles;
  UINTN cnt;

  Status = gBS->LocateHandleBuffer(ByProtocol, &UsbIoProtocol, NULL, &cnt, &handles);
  switch (Status) {
    case EFI_SUCCESS:
      for (int i = 0; i < cnt; i++) {
          gBS->ConnectController(handles[i], NULL, NULL, FALSE);
      }
      gBS->FreePool(handles);
      break;
    case EFI_NOT_FOUND:
      Print(L"No USB devices found\n");
      break;
    default:
      Print(L"Error searching for USB: %ld\n", Status);
  }

  return Status;
}

/**
Ax88772 driver entry point.

@param [in] ImageHandle       Handle for the image.
@param [in] pSystemTable      Address of the system table.

@retval EFI_SUCCESS           Image successfully loaded.

**/
EFI_STATUS
EFIAPI
ax88772_init (
  IN EFI_HANDLE ImageHandle,
  IN EFI_SYSTEM_TABLE * pSystemTable
  )
{
  EFI_STATUS    Status;

  //
  //  Add the driver to the list of drivers
  //
  /*Status = EfiLibInstallDriverBindingComponentName2 (
             ImageHandle,
             pSystemTable,
             &gDriverBinding,
             ImageHandle,
             &gComponentName,
             &gComponentName2
             );*/

  gDriverBinding.ImageHandle = ImageHandle;
  gDriverBinding.DriverBindingHandle = ImageHandle;

  Status = gBS->InstallMultipleProtocolInterfaces(
                &gDriverBinding.DriverBindingHandle,
                &DriverBindingProtocol, gDriverBinding,
                &ComponentNameProtocol, gComponentName,
                &ComponentName2Protocol, gComponentName2,
                NULL);

  if ( !EFI_ERROR ( Status )) {
    DEBUG(D_INFO, L"Installed: DriverBindingProtocol on 0x%08x\r\n", ImageHandle);
    DEBUG(D_INFO, L"Installed: ComponentNameProtocol on 0x%08x\r\n", ImageHandle);
    DEBUG(D_INFO, L"Installed: ComponentName2Protocol on 0x%08x\r\n", ImageHandle);
  }

  if (EFI_ERROR(ax88772_install(ImageHandle, pSystemTable))) {
    Print(L"Failed to install ax88772 driver\n");
  }
  return Status;
}
