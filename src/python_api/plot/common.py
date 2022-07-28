from matplotlib.figure import Figure
import io
import base64
from src.python_api.api.log import log


def savefig_to_b64(fig: Figure):
    log.debug('Saving image to base64')
    buf = io.BytesIO()
    fig.savefig(buf, format='png')
    buf.seek(0)
    return base64.b64encode(buf.read())