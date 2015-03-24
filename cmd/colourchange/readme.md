##colourchange

###installation

`go install github.com/draaglom/GleepostAPI/cmd/colourchange`

###usage

Ensure you are on the `nerdnation-app` branch.

To set all the colours at once:

`colourchange /path/to/GleepostIOS/messaging/AppearanceHelper.m FEFEFE #efefef ffffff E12345 001000 222222`

The six hex colour arguments are:

1. Primary colour

2. Left nav-bar element colour

3. Right nav-bar element colour

4. Nav-bar background colour

5. Campus wall title colour

6. General nav-bar title colour


Alternatively, to change (a) specific colour(s):

```
Usage of colourchange:
  -leftnav="": The left nav-bar colour
  -navbar="": The nav-bar background colour
  -navtitle="": The general title colour
  -primary="": The app's primary colour
  -rightnav="": The right nav-bar colour
  -walltitle="": The campus wall title colour
```

`colourchange -leftnav=eeeeee -primary=deadbe /path/to/GleepostIOS/messaging/AppearanceHelper.m`

Note that using this format, the path to the AppearanceHelper must be the last argument.
