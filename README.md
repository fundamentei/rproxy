## `rproxy`

A proxy that applies AES encryption over requests in order to prevent scrapers from easily accessing our data.

The central idea behind the proxy is that it forwards the requests to the underlying API, encrypts the response and
handles the decryption through WASM. Why WASM? Because nobody knows how to decrypt binary to understand what the
fuck we're doing under the hood—and if they do, they deserve to access the data.

## How do I use it?

1. First you'll need to build the decryption VM

```SH
$ make build-asma shared-key="15365230-aa22-4f5f-aa46-f86076a0b6b2"
```

> The key `15365230-aa22-4f5f-aa46-f86076a0b6b2` will be **_shared_** between the VM and the proxy. It will be used to encrypt all the data and it should be kept in secret. 🤫

2. Configure the proxy. Open `config.toml` and figure out what's good for you. It's documented.
3. Run the proxy!

```SH
$ go run main.go
```

It will listen on `:25259`. You can go ahead and make a request to it using httpie or cURL—whatever. But you can also try to `python3 -m http.server` and open the `index.html` we've put together that shows how to use the VM to the decrypt the proxy responses. Here's everything you need:

```TS
import init, { proxy, build_info } from "./dist/asma/main.js";
(async () => {
  // Initialize the decryption VM
  await init();
  // Prints build information. This is useful to put in Sentry metadata and stuff like that...
  console.log(build_info());

  // This is normally the value of the `Authorization` header that you send to the server
  // to authorize the clients, if you don't want to pass it to the proxy and only keep
  // the `Shared Key`, it's fine. Otherwise, it adds another layer of security by encrypting
  // responses individually with everyone's token.
  const authorization = "";
  const response = await fetch("http://localhost:25256/https://httpbin.org/json");
  // Grab everything that came back from the proxy response as a bytes array
  const bytes = await response.arrayBuffer();
  // ...and send it to the VM for decryption
  console.log(await proxy(new Uint8Array(bytes), authorization));
})();
```

## How does it work?

![The standard request flow](./static/The%20standard%20request%20flow.png)

![The encrypted request flow](./static/The%20encrypted%20request%20flow.png)

## The pitch

> #### Why put a proxy if you can encrypt directly on the API?

That's true. You can. But you should ask yourself the following questions:

1. Are you willing to make the PR across your repositories and deploy that solution straight away?
2. Are you willing to sacrifice the DX of using your regular API and deal with flags for whether or not you should encrypt the response?
3. Do you want to carry over response encryption logic to your existing legacy/already-working-kind-of-thing stuff?

If so...then you're good to go. Otherwise, feel free not to worry about a proxy in front of your existing APIs.

> #### But what about the delay?

Well, we're adding an extra network hop and all the underlying proxying logic by using this solution, however, we need to reasonably think—is it worth my API being exposed to a scraper vs a extra `2ms` delay? Ask yourself this question and make a decision.

My selling point is that we already do bad performing code all the time, and we aren't worried about it. But when it comes to _proxying_ a simple HTTP request to an encryption proxy, we'll "bitch" around a `2ms` delay that it **_could_** add to the request total time?

How do I know that it's `2ms`? Well, imagine a standard network chart—you already have your API exposed to the web, so that's your current latency. By adding a proxy, you're putting your existing API to be called via the internal network, hence, taking all the advantage of security and Gigabit ethernet boards. C'mon, this is solved already. Dicussion's over.

> #### Why WASM? Isn't this too modern?

Already supported everywhere. Fuck IE. My data costs much more than **_less-than-10_** users using that browser that doens't even render me any money—let's face it and play honest.

The reason for it being implemented via WASM is that we'll hide all the encryption logic, so you can't go in the famous "Network Tab" on Chrome, click on "Initiator" and easily figure out my decoding logic and grab the ciphers.
