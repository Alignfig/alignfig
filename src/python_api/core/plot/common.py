from matplotlib.figure import Figure
import matplotlib.pyplot as plt
import io
import base64
from core.api.log import log


plt.rcParams['svg.fonttype'] = 'none'
plt.rcParams['pdf.fonttype'] = 42
plt.rcParams['ps.fonttype'] = 42

def savefig_to_b64(fig: Figure, format: str):
    log.debug('Saving image to base64')
    buf = io.BytesIO()
    fig.savefig(buf, format=format)
    buf.seek(0)
    return base64.b64encode(buf.read())