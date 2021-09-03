# loupedeck

This provides somewhat minimal support for talking to a [Loupedeck
Live](https://loupedeck.com/us/products/loupedeck-live/) from Go.  Supported features:

- Reacting to button, knob, and touchscreen events.
- Displaying images on any of the 3 displays.

In addition, widgets and convienence functions are provided for a
couple higher-level input abstractions.

## Sample code

```
	l, err := loupedeck.Connect("ws://100.127.1.1")
	if err != nil {
		panic(err)
	}

	light1 := loupedeck.NewWatchedInt(0)
	light1.AddWatcher(func (i int) { fmt.Printf("DMX 1->%d\n", i) })
	light2 := loupedeck.NewWatchedInt(0)
	light2.AddWatcher(func (i int) { fmt.Printf("DMX 3->%d\n", i) })
	light3 := loupedeck.NewWatchedInt(0)
	light3.AddWatcher(func (i int) { fmt.Printf("DMX 5->%d\n", i) })

        // Use the left display and the 3 left knobs to adjust 3 independent lights between 0 and 100.
	// Whenever these change, the callbacks from 'AddWatcher' (above) will be called.
	l.NewTouchDial(loupedeck.DisplayLeft, light1, light2, light3, 0, 100)
	
	// Define the 'Circle' button (bottom left) to function as an "off" button for lights 1-3.
	// Similar to NewTouchDial, the callbacks from `AddWatcher` will be called.  This
	// includes an implicit call to the TouchDial's Draw() function, so just calling 'Set'
	// will update the values, the lights (if the callbacks above actually did anything useful),
	// and the Loupedeck.
	
	l.BindButton(loupedeck.Circle, func (b loupedeck.Button, s loupedeck.ButtonStatus){
		light1.Set(0)
		light2.Set(0)
		light3.Set(0)
	})
		
	l.Listen()
```