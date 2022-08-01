import base64
import json
from core.variables import result_image_json

def decode_image_to_file(path: str, image_path: str):
    with open(path) as f:
        data = json.load(f)

    with open(image_path, "wb") as fh:
        fh.write(base64.decodebytes(data[result_image_json].encode()))
