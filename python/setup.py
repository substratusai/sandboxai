from setuptools import setup

# This is needed to force platform specific wheels
setup(has_ext_modules=lambda: True)
