#!/usr/bin/with-contenv bashio
# ==============================================================================
# Home Assistant Community Add-on: Frigate B2 Uploader
# Runs the Frigate B2 Uploader
# ==============================================================================

# Export configuration options as environment variables
export TZ=$(bashio::config 'TZ')
export FRIGATE_IP_ADDRESS=$(bashio::config 'FRIGATE_IP_ADDRESS')
export FRIGATE_PORT=$(bashio::config 'FRIGATE_PORT')
export STORAGE_BACKENDS=$(bashio::config 'STORAGE_BACKENDS')
export B2_REGION=$(bashio::config 'B2_REGION')
export B2_ENDPOINT=$(bashio::config 'B2_ENDPOINT')
export B2_ACCESS_KEY_ID=$(bashio::config 'B2_ACCESS_KEY_ID')
export B2_SECRET_ACCESS_KEY=$(bashio::config 'B2_SECRET_ACCESS_KEY')
export B2_BUCKET_NAME=$(bashio::config 'B2_BUCKET_NAME')

# Run the Frigate B2 Uploader
exec /usr/bin/frigate-b2-uploader
