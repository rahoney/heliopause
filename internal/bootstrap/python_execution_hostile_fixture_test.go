//go:build linux && realcorpus

package bootstrap

// Function-local imports and actions keep current inspection separate from the
// selected hostile body. Raw bytes still require real whole approval each time.
const pythonHostileModule = `def main():
    import os
    import sys
    mode = sys.argv[1]
    if mode == 'outside-read':
        os.write(1, b'fs-ready\n')
        try:
            with open('/home/haa-python-boundary-probe', 'rb') as source:
                source.read(1)
        except OSError:
            pass
    elif mode == 'network':
        import socket
        os.write(1, b'network-ready\n')
        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as connection:
            try:
                connection.connect(('127.0.0.1', 9))
            except OSError:
                pass
    elif mode == 'clone-exec':
        import subprocess
        os.write(1, b'clone-ready\n')
        subprocess.run([sys.executable, '-I', '-S', '-B', '-c',
                        "import os; os.write(1, b'child-body\\n'); open('/proc/self/status', 'rb').read(1)"],
                       check=False)
    elif mode == 'comm-spoof':
        import _thread
        _thread.set_name('haa_gvisor_obse')
        os.write(1, b'spoof-ready\n')
        try:
            with open('/proc/self/status', 'rb') as source:
                source.read(1)
        except OSError:
            pass
    elif mode == 'readonly':
        import errno
        try:
            with open(__file__, 'wb') as destination:
                destination.write(b'unexpected-write')
        except OSError as failure:
            if failure.errno not in (errno.EROFS, errno.EACCES, errno.EPERM):
                raise
            os.write(1, b'readonly-denied\n')
        else:
            return 37
    else:
        raise ValueError('unknown controlled hostile case')
`
