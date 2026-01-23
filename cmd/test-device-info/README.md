# Device Info Test

A standalone test program to verify FRITZ!Box connectivity and device information retrieval.

## Usage

1. **Ensure `.env` is configured** with your FRITZ!Box credentials:
   ```bash
   cp ../../.env.example ../../.env
   # Edit .env with your credentials
   ```

2. **Build the test program:**
   ```bash
   go build -o test-device-info
   ```

3. **Run the test:**
   ```bash
   ./test-device-info
   ```

## Expected Output

The test will:
1. Load configuration from `.env`
2. Connect to your FRITZ!Box
3. Discover all available TR-064 services
4. Retrieve device information (model, firmware, serial, etc.)
5. Display the results in human-readable format and JSON

## Example Output

```
=== FRITZ!Box MCP Server - Device Info Test ===

1. Loading configuration from .env...
   ✓ Configured for: http://fritz.box:49000

2. Creating TR-064 client...
   ✓ Client created

3. Discovering services from FRITZ!Box...
   ✓ Discovered 25 services

4. Available services:
   - urn:dslforum-org:service:DeviceInfo:1
   - urn:dslforum-org:service:WANIPConnection:1
   ...

5. Retrieving device information...

=== Device Information ===

Manufacturer        : AVM
Model              : FRITZ!Box 7590
Serial Number      : 1234567890AB
Software Version   : 154.07.57
Hardware Version   : FRITZ!Box 7590
Spec Version       : 1.0
Uptime (seconds)   : 234567

=== Full Response (JSON) ===

{
  "NewManufacturerName": "AVM",
  "NewModelName": "FRITZ!Box 7590",
  ...
}

✓ Test completed successfully!
```

## Troubleshooting

- **Config error**: Verify `.env` exists and contains valid credentials
- **Connection error**: Ensure your FRITZ!Box is reachable at the configured URL
- **Auth error**: Check username and password in `.env`
- **Service not found**: Your FRITZ!Box model may use different service names
