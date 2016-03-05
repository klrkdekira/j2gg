# j2gg
JSON2GeocodeGeoJSON

Convert any json line stream to Google Geocoder and output GeoJSON.
This violates the Google Geocoder EULA, not suitable for businesses.

## Usage

`go get -u github.com/klrkdekira/j2gg`

`cat line.json | j2gg -keys "attributes,to,be,geocoded"`
