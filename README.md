# j2gg
JSON2GeocodeGeoJSON

Convert any json line stream.

## Usage

`go get -u github.com/klrkdekira/j2gg`

`cat line.json | j2gg -keys "address,state" -geocoder "http://maps.googleapis.com/maps/api/geocode/json?address=" -lat "results,0,geometry,location,lat" -lng "results,0,geometry,location,lng"