from flask import Flask, request, jsonify, json
from src.python_api.api.plot import generate_figure_from_json
from src.python_api.variables import result_image_json, encoding
from src.python_api.api.log import log


app = Flask(__name__)
logger = app.logger
@app.route("/generate_fig", methods=['POST'])
def generate_fig():
    data = json.loads(json.htmlsafe_dumps(request.get_json()))
    log.debug(f'Data from request: "{data}"')
    figure = generate_figure_from_json(data)
    response = {result_image_json: figure.decode(encoding)}
    return jsonify(response)

if __name__ == '__main__':
    app.run(host='127.0.0.1',debug=True)
