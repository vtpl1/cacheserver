# Cache server

1. Existing endpoint access

1. Moving to websocket => No reconnection
1. Streaming the data over websocket => Streaming data population
1. Limiting total number of records to be served. ( Max 200 data points)
1. Intermediate cache to serve data upto 3 months very efficiently.
1. Snap to nearest intersting point, (Person, Vehicle, Event)
1. When zoomed out > 2 then the debouncing time should be more > 5 sec



Message:
1. commandId
1. pivotPoint
1. displayMax
1. displayMin
1. domainMax
1. domainMin


```ts
type Command = {
  commandId: string;
  pivotPoint: number;
  displayMin: number;
  displayMax: number;
  domainMin: number;
  domainMax: number;
};
```


