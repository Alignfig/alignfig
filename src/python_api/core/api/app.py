from flask import Flask, request, jsonify, json, Response
from json import JSONDecodeError
from .plot import generate_figure_from_json
from core.variables import result_image_json, result_error, encoding, \
    request_alignment_type, request_alignment_format
from .log import log
import traceback


app = Flask(__name__)
logger = app.logger
@app.route("/generate_fig", methods=['POST'])
def generate_fig():
    try:
        data = json.loads(json.htmlsafe_dumps(request.get_json()))
    except JSONDecodeError as err:
        return handle_error(err)

    log.debug(f'Data from request: "{data[request_alignment_format]}, {data[request_alignment_type]}"')

    try:
        figure = generate_figure_from_json(data)
        response = {result_image_json: figure.decode(encoding)}
    except ValueError as err:
        return handle_error(err)

    return jsonify(response)


def handle_error(err: Exception) -> Response:
    logger.error(traceback.format_exc())
    return jsonify({result_error: err, result_image_json: ""})


if __name__ == '__main__':
    app.run(host='127.0.0.1',debug=True)
