from flask.cli import FlaskGroup
import click

from core.api.app import app
from core.util.image_decoder import decode_image_to_file

cli = FlaskGroup(app)

@cli.command("decode_image")
@click.argument("path")
@click.argument("image_path")
def decode_image(path, image_path):
    decode_image_to_file(path=path, image_path=image_path)

if __name__ == "__main__":
    cli()