"use strict";

function defaultAdapter(baseUrl) {
  function request(url, data) {
    var req = new XMLHttpRequest();
    req.open(data ? "POST" : "GET", "https://cors-anywhere.herokuapp.com/" + url);
    if (data) {
      req.setRequestHeader("Content-Type", "application/json");
      req.send(data);
    } else {
      req.send();
    }
    return req;
  };

  return {
    create: function(data, callback) {
      try {
        var req = request(baseUrl + "/api/secret", JSON.stringify(data));
        req.onerror = function(event) {
          callback(event.target);
        };
        req.addEventListener("loadend", function(event) {
          callback(null, JSON.parse(event.target.responseText));
        });
      } catch (e) {
        callback(e);
      }
    },
    get: function(mnemo, callback) {
      var req = request(baseUrl + "/api/secret/" + mnemo);
      req.onerror = function(event) {
        callback("keyDecodeFail");
      };
      req.addEventListener("loadend", function(event) {
        if (event.target.status !== 200) {
          return callback("secretNotFound")
        }
        callback(null, JSON.parse(event.target.responseText));
      });
    }
  };
};

function pastebinAdapter(apiOptions) {
  function buildParams(data) {
    return Object.keys(data).map(function(key) {
      return key + "=" + encodeURIComponent(data[key]);
    });
  }
  function request(url, data) {
    var req = new XMLHttpRequest();
    req.open(data ? "POST" : "GET", "https://cors-anywhere.herokuapp.com/" + url);
    if (data !== undefined) {
      req.setRequestHeader("Content-Type", "application/x-www-form-urlencoded");
      var postData =
        buildParams(data)
        .concat(buildParams(apiOptions))
        .join("&");
      req.send(postData);
    } else {
      req.send();
    }
    return req;
  }

  return {
    create: function(data, callback) {
      try {
        var req = request("https://pastebin.com/api/api_post.php", {
          api_paste_code: JSON.stringify(data),
          api_option: "paste"
        });
        req.onerror = function(event) {
          callback(event.target);
        };
        req.addEventListener("loadend", function(event) {
          callback(null, {
            mnemo: event.target.responseText.replace("https://pastebin.com", "https://pastebin.com/raw")
          });
        });
      } catch (e) {
        callback(e);
      }
    },
    get: function(url, callback) {
      try {
        var req = request(url);
        req.onerror = function(event) {
          callback("keyDecodeFail");
        };
        req.addEventListener("loadend", function(event) {
          if (event.target.status !== 200) {
            return callback("secretNotFound")
          }
          var json = JSON.parse(event.target.responseText);
          req = request("https://pastebin.com/api/api_post.php", {
            api_option: "delete",
            api_paste_key: url.replace("https://pastebin.com/raw/", "")
          });
          req.onerror = function(event) {
            callback("keyDecodeFail");
          };
          req.addEventListener("loadend", function(event) {
            callback(null, json);
          });
        });
      } catch (e) {
        callback(e);
      }
    }
  }
}

/*var api = pastebinAdapter({
  api_paste_private: 1, // unlisted
  api_paste_expire_date: "10M",
  api_dev_key: "54c47562c67620c1c5cde10dad1c2b89",
  api_user_key: "a91facd26a9c74577d4e4a43c2ce409a"
});*/

var api = defaultAdapter("https://keedrop.com")

function transferEncode(value) {
  return window.nacl.util.encodeBase64(value).replace(/\//g, "!");
}

function transferDecode(value) {
  return window.nacl.util.decodeBase64(value.replace(/!/g, "/"));
}

var ERRORS = {
  "keyDecodeFail": "Could not interpret your key. Check that the link you clicked is complete.",
  "noSecretId": "It looks like someone tampered with your link.",
  "secretNotFound": "Could not retrieve secret. It has either expired (24h) or someone else already retrieved it.",
  "decryptionFailed": "Oops. Could not decrypt the data.",
  "sendFail": "Could not connect to server to store encrypted data."
};

function copyToClipboard(source, callback) {
  function fallback() {
    try {
      source.select();
      source.setSelectionRange(0, 9999);
      callback(document.execCommand("copy"));
    } catch (e) {
      callback(false);
    }
  }
  // Only works on HTTPS served sites
  if (!navigator.clipboard) {
    return fallback();
  }
  navigator.clipboard.writeText(source.value)
    .then(callback.bind(this, true), callback.bind(this, false));
}

function onCopyClick(event) {
  event.preventDefault && event.preventDefault();
  copyToClipboard(document.getElementById("resultBox"), function(success) {
    event.srcElement.innerText = "Copied";
  });
  return false;
}

function hideError() {
  var errorDiv = document.querySelector(".error");
  errorDiv.classList.remove("reveal");
}

function showError(messageId, hideContent) {
  var errorDiv = document.querySelector(".error");
  errorDiv.innerHTML = ERRORS[messageId];
  errorDiv.classList.add("reveal");
  if (hideContent) {
    var content = document.querySelector(".content");
    content.classList.add("hide");
  }
}

function onEncryptSubmit(event) {
  // Cancel default form submit handling
  event.preventDefault && event.preventDefault();

  var form = event.currentTarget;
  var secret = window.nacl.util.decodeUTF8(form.elements.secret.value);
  var keyPair = window.nacl.box.keyPair();
  var nonce = window.nacl.randomBytes(window.nacl.box.nonceLength);
  var encrypted = window.nacl.box(secret, nonce, keyPair.publicKey, keyPair.secretKey);

  hideError();

  var resultBox = document.getElementById("resultBox");
  resultBox.value = "";
  var button = form.querySelector("button");
  button.disabled = true;

  api.create({
    pubkey: transferEncode(keyPair.publicKey),
    nonce: transferEncode(nonce),
    secret: transferEncode(encrypted)
  }, function(error, result) {
    button.disabled = false;
    if (error) {
      return showError("sendFail");
    }
    var decodeLink = location.protocol + "//" + location.host + "/r#" + result.mnemo + "_" + transferEncode(keyPair.secretKey);
    resultBox.value = decodeLink;
    resultBox.parentNode.parentNode.parentNode.classList.add("reveal");
    resultBox.focus();
    resultBox.select();
    // For mobile devices
    resultBox.setSelectionRange(0, 9999);
  });
  return false;
}

function onDecryptClicked(resultBox) {
  if (location.hash.indexOf("_") == -1) {
    showError("noSecretId");
    return;
  }

  var components = location.hash.substr(1).split("_");
  var mnemo = components[0];
  var secretKey = components[1];

  try {
    var decodedSecretKey = transferDecode(secretKey);
    if (decodedSecretKey.length != 32) {
      throw new Error("Key length mismatch");
    }
    api.get(mnemo, function(error, result) {
      if (error) {
        return showError("keyDecodeFail", true);
      }
      var plainText = window.nacl.box.open(
        transferDecode(result.secret),
        transferDecode(result.nonce),
        transferDecode(result.pubkey),
        decodedSecretKey);
      if (plainText) {
        resultBox.value = window.nacl.util.encodeUTF8(plainText);
        resultBox.parentNode.parentNode.parentNode.classList.add("reveal");
      } else {
        showError("decryptionFailed");
      }
    });
  } catch(e) {
    showError("keyDecodeFail", true);
  }
}

function initReadPage() {
  var button = document.querySelector("button:disabled");
  button.disabled = false;
  var resultBox = document.getElementById("resultBox");
  button.onclick = onDecryptClicked.bind(document, resultBox);
}

function initStorePage() {
  var button = document.querySelector("button:disabled");
  button.disabled = false;
  var form = document.getElementById("storeForm");
  form.onsubmit = onEncryptSubmit;
}

window.addEventListener("click", function(event) {
  var source = event.srcElement;
  // E-Mail spam protection
  if (source.tagName === "A" && source.dataset.name && source.dataset.domain && source.dataset.tld) {
    window.location.href = "mailto:" + source.dataset.name + "@" + source.dataset.domain + "." + source.dataset.tld;
    event.preventDefault();
    return false;
  } else if (source.id === "copy") {
    return onCopyClick(event);
  }
})

window.addEventListener("load", function() {
  if (document.getElementById("encrypt") != null) {
    initStorePage();
  } else if (document.getElementById("decrypt") != null) {
    initReadPage();
  }
});
