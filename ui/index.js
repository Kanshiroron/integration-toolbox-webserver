// this function makes sure that form are in correct state if the page is reloaded
// (since most browser do not reset fields values).
function bodyOnLoad() {
  // database
  databaseDriverChanged();

  // request
  requestURLChanged(document.getElementById("requestURL").value);
  requestTLSInsecureChanged(document.getElementById("requestTLSInsecure").checked);
  requestProxyChanged(document.getElementById("requestProxyEnable").checked);

  // tcp
  tcpEchoBodyChanged(document.getElementById("tcpEchoBody").checked);
  tcpTLSEnabledChanged(document.getElementById("tcpTLSEnable").checked);
  tcpTLSInsecureChanged(document.getElementById("tcpTLSInsecure").checked);
}

// crash
function crash(button, resultP) {
  clearOldResult(button, resultP);

  // building URL
  let url = "/crash";
  let queryParams = new Array();
  // exit code
  let crashExitCode = document.getElementById("crashExitCode").value;
  if (crashExitCode.length > 0) {
    if (parseInt(crashExitCode) < 0) {
      resultError(resultP, "exit code inferior to 0", button);
      return;
    }
    queryParams.push("code=" + parseInt(crashExitCode));
  }
  // frequency
  var timeout = document.getElementById("crashTimeout").value;
  if (timeout.length != 0) {
    queryParams.push("timeout=" + timeout.trim());
  }
  // query params
  if (queryParams.length > 0) {
    url += "?" + queryParams.join("&")
  }

  // building request
  let xhr = new XMLHttpRequest();
  xhr.onreadystatechange = function() {
    if (this.readyState == 4) {
      if (this.readyState == 4) {
        if (this.status == 200) {
          resultOk(resultP, "ok, server will crash in a second", button);
        } else if (this.status != 0) {
          resultError(resultP, this.responseText, button);
        } else {
          resultConnectionError(resultP, button);
        }
      }
    }
  };

  // sending request
  xhr.open("GET", url, true);
  xhr.send();
}

// download
function download(button, resultP) {
  clearOldResult(button, resultP);

  // building URL
  let url = "/download";
  let downloadSize = document.getElementById("downloadSize").value;
  if (downloadSize.length > 0) {
    if (parseInt(downloadSize) < 0) {
      resultError(resultP, "download size is negative", button);
      return;
    }
    url += "?size=" + parseInt(downloadSize);
  } else {
    downloadSize = 1024*1024;
  }

  // progress bar
  var progressBar = document.getElementById("downloadProgressBar")
  progressBar.style.display = "block";
  var progressBarProgress = document.getElementById("downloadProgressBarProgress")
  progressBarProgress.style.width = "0%";
  progressBarProgress.innerHTML = "0%";

  // building request
  var startDate = new Date();
  let xhr = new XMLHttpRequest();
  xhr.onprogress = function(event) {
    if (event.lengthComputable) {
      let percent = Math.round((event.loaded / event.total) * 100);
      progressBarProgress.style.width = percent + "%";
      progressBarProgress.innerHTML = percent + "%";
    }
  };
  xhr.onreadystatechange = function() {
    if (this.readyState == 4) {
      progressBar.style.display = "none";
      if (this.status == 200) {
        let endDate = new Date();
        let duration = Math.round((endDate - startDate) / 100) / 10;
        let speed = Math.round(downloadSize / (endDate - startDate) * 1000);
        resultOk(resultP, sizeToString(downloadSize) + " downloaded in "+ duration + " seconds (" + sizeToString(speed) + "/s)", button);
      } else if (this.status != 0) {
        resultError(resultP, this.responseText, button);
      } else {
        resultConnectionError(resultP, button);
      }
    }
  };

  // sending request
  xhr.open("GET", url, true);
  xhr.send();
}

// sleep
function sleep(button, resultP) {
  clearOldResult(button, resultP);

  // building URL
  let url = "/sleep";
  let queryParams = new Array();
  // duration
  var sleepDuration = document.getElementById("sleepDuration").value;
  if (sleepDuration.length > 0) {
    queryParams.push("duration=" + sleepDuration.trim());
  } else {
    sleepDuration = "1s"
  }
  // status code
  var sleepStatusCodeInt = 200;
  let sleepStatusCode = document.getElementById("sleepStatusCode").value;
  if (sleepStatusCode.length != 0) {
    sleepStatusCodeInt = parseInt(sleepStatusCode);
    if ((sleepStatusCodeInt < 100) || (sleepStatusCodeInt > 599)) {
      resultError(resultP, "status code out of bounds (100 - 599)", button);
      return;
    }
    queryParams.push("code=" + sleepStatusCodeInt);
  }
  // query params
  if (queryParams.length > 0) {
    url += "?" + queryParams.join("&")
  }

  // building request
  var startDate = new Date();
  let xhr = new XMLHttpRequest();
  xhr.onreadystatechange = function() {
    if (this.readyState == 4) {
      if (this.status == sleepStatusCodeInt) {
        resultOk(resultP, "server slept for " + sleepDuration, button);
      } else if (this.status != 0) {
        resultError(resultP, this.responseText, button);
      } else {
        resultConnectionError(resultP, button);
      }
    }
  };

  // sending request
  xhr.open("GET", url, true);
  xhr.send();
}

// status code
function statusCode(button, resultP) {
  clearOldResult(button, resultP);

  // building URL
  let url = "/status_code?";
  // status code
  var statusCodeInt = 200;
  let statusCode = document.getElementById("statusCodeCode").value;
  if (statusCode.length != 0) {
    statusCodeInt = parseInt(statusCode);
    if ((statusCodeInt < 100) || (statusCodeInt > 599)) {
      resultError(resultP, "status code out of bounds (100 - 599)", button);
      return;
    }
    url += "code=" + statusCodeInt;
  }

  // building request
  let xhr = new XMLHttpRequest();
  xhr.onreadystatechange = function() {
    if (this.readyState == 4) {
      if (this.status == statusCodeInt) {
        resultOk(resultP, "server responded with correct status code " + statusCodeInt, button);
      } else if (this.status != 0) {
        resultError(resultP, this.responseText, button);
      } else {
        resultConnectionError(resultP, button);
      }
    }
  };

  // sending request
  xhr.open("GET", url, true);
  xhr.send();
}

// upload
function upload(button, resultP) {
  clearOldResult(button, resultP);

  // check file selected
  if (document.getElementById("uploadFile").files.length == 0) {
    resultError(resultP, "no file selected", button);
    return;
  }

  // building URL
  let url = "/upload";

  // progress bar
  var progressBar = document.getElementById("uploadProgressBar")
  progressBar.style.display = "block";
  var progressBarProgress = document.getElementById("uploadProgressBarProgress")
  progressBarProgress.style.width = "0%";
  progressBarProgress.innerHTML = "0%";

  // file size
  let file = document.getElementById("uploadFile").files[0];
  var fileSize = file.size;

  // building request
  var startDate = new Date();
  let xhr = new XMLHttpRequest();
  xhr.upload.addEventListener("progress", function(event) {
    let percent = Math.round((event.loaded / fileSize) * 100);
    progressBarProgress.style.width = percent + "%";
    progressBarProgress.innerHTML = percent + "%";
  }, false);
  xhr.onreadystatechange = function() {
    if (this.readyState == 4) {
      progressBar.style.display = "none";
      if (this.status == 200) {
        let endDate = new Date();
        let duration = Math.round((endDate - startDate) / 100) / 10;
        let speed = Math.round(fileSize / (endDate - startDate) * 1000);
        resultOk(resultP, sizeToString(fileSize) + " uploaded in "+ duration + " seconds (" + sizeToString(speed) + "/s)", button);
      } else if (this.status != 0) {
        resultError(resultP, this.responseText, button);
      } else {
        resultConnectionError(resultP, button);
      }
    }
  };

  // form data
  let formData = new FormData();
  formData.append("file", file);

  // sending request
  xhr.open("POST", url, true);
  xhr.send(formData);
}

// CPU load
function cpuLoad(button, resultP) {
  clearOldResult(button, resultP);

  // building URL
  let url = "/cpu/load";
  let queryParams = new Array();
  // nb threads
  let nbThreads = document.getElementById("cpuLoadNbThreads").value;
  if (nbThreads.length != 0) {
    if (parseInt(nbThreads) < 0) {
      resultError(resultP, "number of threads inferior to 0", button);
      return;
    }
    queryParams.push("nb_threads=" + nbThreads.trim());
  }
  // timeout
  var timeout = document.getElementById("cpuLoadTimeout").value;
  if (timeout.length != 0) {
    queryParams.push("timeout=" + timeout.trim());
  }
  // query params
  if (queryParams.length > 0) {
    url += "?" + queryParams.join("&")
  }

  // building request
  let xhr = new XMLHttpRequest();
  xhr.onreadystatechange = function() {
    if (this.readyState == 4) {
      if (this.status == 200) {
        resultOk(resultP, "server CPU loaded", button);
      } else if (this.status != 0) {
        resultError(resultP, this.responseText, button);
      } else {
        resultConnectionError(resultP, button);
      }
    }
  };

  // sending request
  xhr.open("GET", url, true);
  xhr.send();
}

// CPU reset
function cpuReset(button, resultP) {
  clearOldResult(button, resultP);

  // building URL
  let url = "/cpu/reset";

  // building request
  let xhr = new XMLHttpRequest();
  xhr.onreadystatechange = function() {
    if (this.readyState == 4) {
      if (this.status == 200) {
        resultOk(resultP, "server CPU load reset", button);
      } else if (this.status != 0) {
        resultError(resultP, this.responseText, button);
      } else {
        resultConnectionError(resultP, button);
      }
    }
  };

  // sending request
  xhr.open("GET", url, true);
  xhr.send();
}

// RAM increase
function ramIncrease(button, resultP) {
  clearOldResult(button, resultP);

  // building URL
  let url = "/ram/increase";
  let increaseSize = document.getElementById("ramIncreaseSize").value;
  if (increaseSize.length > 0) {
    if (parseInt(increaseSize) <= 0) {
      resultError(resultP, "memory increase size must be strictly positive", button);
      return;
    }
    url += "?size=" + parseInt(increaseSize);
  }

  // building request
  let xhr = new XMLHttpRequest();
  xhr.onreadystatechange = function() {
    if (this.readyState == 4) {
      if (this.status == 200) {
        resultOk(resultP, "server memory increased<br />"+this.responseText, button);
      } else if (this.status != 0) {
        resultError(resultP, this.responseText, button);
      } else {
        resultConnectionError(resultP, button);
      }
    }
  };

  // sending request
  xhr.open("GET", url, true);
  xhr.send();
}

// RAM decrease
function ramDecrease(button, resultP) {
  clearOldResult(button, resultP);

  // building URL
  let url = "/ram/decrease";
  let decreaseSize = document.getElementById("ramDecreaseSize").value;
  if (decreaseSize.length > 0) {
    if (parseInt(decreaseSize) < 0) {
      resultError(resultP, "memory decrease size is negative", button);
      return;
    }
    url += "?size=" + parseInt(decreaseSize);
  }

  // building request
  let xhr = new XMLHttpRequest();
  xhr.onreadystatechange = function() {
    if (this.readyState == 4) {
      if (this.status == 200) {
        resultOk(resultP, "server memory decreased<br />"+this.responseText, button);
      } else if (this.status == 206) { // not fully released
        resultWarning(resultP, this.responseText.replace(/(?:\r\n|\r|\n)/g, '<br>'), button);
      } else if (this.status != 0) {
        resultError(resultP, this.responseText, button);
      } else {
        resultConnectionError(resultP, button);
      }
    }
  };

  // sending request
  xhr.open("GET", url, true);
  xhr.send();
}

// RAM leak
function ramLeak(button, resultP) {
  clearOldResult(button, resultP);

  // building URL
  let url = "/ram/leak";
  let queryParams = new Array();
  // size
  var ramLeakSize = document.getElementById("ramLeakSize").value;
  if (ramLeakSize.length != 0) {
    queryParams.push("size=" + ramLeakSize.trim());
  }
  // frequency
  var ramLeakFrequency = document.getElementById("ramLeakFrequency").value;
  if (ramLeakFrequency.length != 0) {
    queryParams.push("frequency=" + ramLeakFrequency.trim());
  }
  // query params
  if (queryParams.length > 0) {
    url += "?" + queryParams.join("&")
  }

  // building request
  let xhr = new XMLHttpRequest();
  xhr.onreadystatechange = function() {
    if (this.readyState == 4) {
      if (this.status == 200) {
        resultOk(resultP, "server memory leak triggered", button);
      } else if (this.status != 0) {
        resultError(resultP, this.responseText, button);
      } else {
        resultConnectionError(resultP, button);
      }
    }
  };

  // sending request
  xhr.open("GET", url, true);
  xhr.send();
}

// RAM reset
function ramReset(button, resultP) {
  clearOldResult(button, resultP);

  // building URL
  let url = "/ram/reset";

  // building request
  let xhr = new XMLHttpRequest();
  xhr.onreadystatechange = function() {
    if (this.readyState == 4) {
      if (this.status == 200) {
        resultOk(resultP, "server memory leak reset<br />"+this.responseText, button);
      } else if (this.status != 0) {
        resultError(resultP, this.responseText, button);
      } else {
        resultConnectionError(resultP, button);
      }
    }
  };

  // sending request
  xhr.open("GET", url, true);
  xhr.send();
}

// RAM status
function ramStatus(button, resultP) {
  clearOldResult(button, resultP);

  // building URL
  let url = "/ram/status";

  // building request
  let xhr = new XMLHttpRequest();
  xhr.onreadystatechange = function() {
    if (this.readyState == 4) {
      if (this.status == 200) {
        resultOk(resultP, this.responseText, button);
      } else {
        resultConnectionError(resultP, button);
      }
    }
  };

  // sending request
  xhr.open("GET", url, true);
  xhr.send();
}

// started
function startedGet(button, resultP) {
  monitoringGet(button, resultP, "/started");
}
function startedPost(button, resultP) {
  monitoringPost(button, resultP, "/started");
}

// alive
function aliveGet(button, resultP) {
  monitoringGet(button, resultP, "/alive");
}
function alivePost(button, resultP) {
  monitoringPost(button, resultP, "/alive");
}

// alive
function readyGet(button, resultP) {
  monitoringGet(button, resultP, "/ready");
}
function readyPost(button, resultP) {
  monitoringPost(button, resultP, "/ready");
}

// database
function databaseConnect(button, resultP) {
  clearOldResult(button, resultP);

  // building URL
  let url = "/database/connect";

  // form data
  let formData = databaseFormData();

  // building request
  let xhr = new XMLHttpRequest();
  xhr.onreadystatechange = function() {
    if (this.readyState == 4) {
      if (this.status == 200) {
        resultOk(resultP, "server successfuly connected to the database", button);
      } else if (this.status != 0) {
        resultError(resultP, this.responseText, button);
      } else {
        resultConnectionError(resultP, button);
      }
    }
  };

  // sending request
  xhr.open("POST", url, true);
  xhr.send(formData);
}

// database
function databaseQuery(button, resultP) {
  clearOldResult(button, resultP);

  // building URL
  let url = "/database/query";

  // form data
  let formData = databaseFormData();
  // query
  let query = document.getElementById("databaseQueryQuery").value;
  if (query.length == 0) {
    resultError(resultP, "empty query", button);
    return;
  }
  formData.append("query", query);

  // building request
  let xhr = new XMLHttpRequest();
  xhr.onreadystatechange = function() {
    if (this.readyState == 4) {
      if (this.status == 200) {
        resultOk(resultP, "server successfuly connected to the database", button);
      } else if (this.status != 0) {
        resultError(resultP, this.responseText, button);
      } else {
        resultConnectionError(resultP, button);
      }
    }
  };

  // sending request
  xhr.open("POST", url, true);
  xhr.send(formData);
}

function databaseDriverChanged() {
  databaseTLSEnableChanged(document.getElementById("databaseConnectionTLSEnable").checked);
}

function databaseTLSEnableChanged(enabled) {
  showFormSection("dbConnectionTLSConfig", enabled);
  if (!enabled) {
    return;
  }
  let dbEngine = document.getElementById("databaseEngine").value;
  let sslMode = document.getElementById("databaseTLSMode").value;
  let tlsInscure = document.getElementById("databaseConnectionTLSInsecure").checked;
  document.getElementById("databaseTLSMode").disabled = (dbEngine != "postgres");
  document.getElementById("databaseConnectionTLSInsecure").disabled = (dbEngine == "postgres");
  document.getElementById("databaseConnectionTLSCA").disabled = ((tlsInscure && (dbEngine != "postgres")) || ((dbEngine == "postgres") && (sslMode == "require")));
  document.getElementById("databaseConnectionTLSUserCert").disabled = (dbEngine == "sqlserver");
  document.getElementById("databaseConnectionTLSUserKey").disabled = (dbEngine == "sqlserver");
}

function databaseTLSModeChanged() {
  databaseTLSEnableChanged(document.getElementById("databaseConnectionTLSEnable").checked);
}

function databaseTLSInsecureChanged(checked) {
  document.getElementById("databaseConnectionTLSCA").disabled = checked;
}

function databaseFormData() {
  let formData = new FormData();
  // engine
  let dbEngine = document.getElementById("databaseEngine").value;
  formData.append("engine", dbEngine);
  // host
  let host = document.getElementById("databaseHost").value;
  if (host.length > 0) {
    formData.append("host", host);
  }
  // port
  let port = document.getElementById("databasePort").value;
  if (port.length > 0) {
    formData.append("port", port);
  }
  // username
  let username = document.getElementById("databaseUsername").value;
  if (username.length > 0) {
    formData.append("username", username);
  }
  // password
  let password = document.getElementById("databasePassword").value;
  if (password.length > 0) {
    formData.append("password", password);
  }
  // database
  let database = document.getElementById("databaseDBName").value;
  if (database.length > 0) {
    formData.append("db_name", database);
  }
  // TLS
  let tlsEnabled = document.getElementById("databaseConnectionTLSEnable").checked;
  if (tlsEnabled) {
    formData.append("tls_enabled", tlsEnabled);
    // ca
    let tlsCA = document.getElementById("databaseConnectionTLSCA").files[0];
    // mode (postgres only)
    if (dbEngine == "postgres") {
      let sslMode = document.getElementById("databaseTLSMode").value
      formData.append("ssl_mode", sslMode);
      if ((sslMode != "require") && tlsCA) {
        formData.append("tls_ca", tlsCA);
      }
    } else {
      let tlsInscure = document.getElementById("databaseConnectionTLSInsecure").value;
      formData.append("tls_insecure", tlsInscure);
      if (!tlsInscure && tlsCA) {
        formData.append("tls_ca", tlsCA);
      }
    }
    // user certificate
    let userCert = document.getElementById("databaseConnectionTLSUserCert").files[0];
    let userKey = document.getElementById("databaseConnectionTLSUserKey").files[0];
    if (userCert && userKey) {
      formData.append("tls_user_cert", userCert);
      formData.append("tls_user_key", userKey);
    }
  }
  return formData;
}

// ping
function ping(button, resultP) {
  clearOldResult(button, resultP);

  // building URL
  let url = "/ping";
  // host
  let host = document.getElementById("pingHost").value;
  if (host == 0) {
    resultError(resultP, "empty hostname or IP", button);
    return;
  }
  url += "?host=" + encodeURI(host);
  // count
  let count = document.getElementById("pingCount").value;
  if (count.length > 0) {
    if (parseInt(count) < 1) {
      resultError(resultP, "ping count is inferior to 1", button);
      return;
    }
    url += "&count=" + parseInt(count);
  }

  // building request
  let xhr = new XMLHttpRequest();
  xhr.onreadystatechange = function() {
    if (this.readyState == 4) {
      if (this.status == 200) {
        resultOk(resultP, "success<br />"+this.responseText, button);
      } else if (this.status != 0) {
        resultError(resultP, this.responseText, button);
      } else {
        resultConnectionError(resultP, button);
      }
    }
  };

  // sending request
  xhr.open("GET", url, true);
  xhr.send();
}

// request
function request(button, resultP) {
  clearOldResult(button, resultP);

  let formData = new FormData();
  // url
  let requestURL = document.getElementById("requestURL").value;
  if (requestURL.length == 0 ){
    resultError(resultP, "empty URL", button);
    return;
  } else if (!httpURLRegexp.test(requestURL) && !websocketURLRegexp.test(requestURL)) {
    resultError(resultP, "URL must start with \"http://\", \"https://\", \"ws://\" or \"wss://\"", button);
    return;
  }
  formData.append("url", requestURL);
  // method
  if (!websocketURLRegexp.test(requestURL)) {
    formData.append("method", document.getElementById("requestMethod").value);
  }
  // connection timeout
  let connectionTimeout = document.getElementById("requestConnectionTimeout").value;
  if (connectionTimeout.length > 0) {
    formData.append("connection_timeout", connectionTimeout);
  }
  // echo headers
  let echoHeaders = document.getElementById("requestEchoHeaders").checked;
  if (echoHeaders) {
      formData.append("echo_headers", echoHeaders);
  }
  // echo body
  let echoBody = document.getElementById("requestEchoBody").checked;
  if (echoBody) {
      formData.append("echo_body", echoBody);
  }

  // tls
  if (requestURL.startsWith('https:\/\/') || requestURL.startsWith('wss:\/\/')) {
    // insecure
    let insecureTLS = document.getElementById("requestTLSInsecure").checked;
    let tlsCA = document.getElementById("requestTLSCA").files[0];
    if (insecureTLS) {
      formData.append("tls_insecure", insecureTLS);
    } else if (tlsCA) {
      formData.append("tls_ca", tlsCA);
    }
    // user certificate
    let userCert = document.getElementById("requestTLSUserCert").files[0];
    let userKey = document.getElementById("requestTLSUserKey").files[0];
    if (userCert && userKey){
      formData.append("tls_user_cert", userCert);
      formData.append("tls_user_key", userKey);
    }
  }

  // proxy
  if (document.getElementById("requestProxyEnable").checked) {
    // url
    let proxyURL = document.getElementById("requestProxyURL").value;
    if (proxyURL.length == 0 ){
      resultError(resultP, "empty proxy URL", button);
      return;
    } else if (!httpURLRegexp.test(proxyURL)) {
      resultError(resultP, "proxy URL must start with \"http://\" or \"https://\"", button);
      return;
    }
    formData.append("proxy_url", proxyURL);
    // username
    let proxyUsername = document.getElementById("requestProxyUsername").value;
    if (proxyUsername.length > 0) {
      formData.append("proxy_username", proxyUsername);
    }
    // password
    let proxyPassword = document.getElementById("requestProxyPassword").value;
    if (proxyPassword.length > 0) {
      formData.append("proxy_password", proxyPassword);
    }
  }

  // building request
  let xhr = new XMLHttpRequest();
  xhr.onreadystatechange = function() {
    if (this.readyState == 4) {
      if (this.status == 200) {
        let answer = escapeHTML(this.responseText.replaceAll(/[\n\r]/g, "##NEW_LINE##")).replaceAll("##NEW_LINE##", "<br />");
        resultOk(resultP, "success<br />"+answer, button);
      } else if (this.status != 0) {
        resultError(resultP, this.responseText, button);
      } else {
        resultConnectionError(resultP, button);
      }
    }
  };

  // sending request
  xhr.open("POST", "/request", true);
  xhr.send(formData);
}

function requestURLChanged(value) {
  showFormSection('requestTLSConfig', value.startsWith('https:\/\/') || value.startsWith('wss:\/\/'));
}

function requestTLSInsecureChanged(checked) {
  document.getElementById("requestTLSCA").disabled = checked;
}

function requestProxyChanged(checked) {
  showFormSection('requestProxyConfig', checked);
}

// tcp
function tcp(button, resultP) {
  clearOldResult(button, resultP);

  let formData = new FormData();
  // host
  let host = document.getElementById("tcpHost").value;
  if (host.length == 0 ){
    resultError(resultP, "empty host", button);
    return;
  } else if (host.includes("://")) {
    resultError(resultP, "the must not contain any scheme (i.e.: \"tcp://\" or equivalent)", button);
    return;
  }
  formData.append("host", host);
  // connection timeout
  let connectionTimeout = document.getElementById("tcpConnectionTimeout").value;
  if (connectionTimeout.length > 0) {
    formData.append("connection_timeout", connectionTimeout);
  }
  // echo body
  let echoBody = document.getElementById("tcpEchoBody").checked;
  if (echoBody) {
      formData.append("echo_body", echoBody);
      // echo body size
      let echoBodySize = document.getElementById("tcpEchoBodySize").checked;
      if (echoBodySize.length > 0) {
        if (parseInt(echoBodySize) <= 0) {
          resultError(resultP, "echo body size can't be inferior or equal to 0", button);
          return;
        }
        formData.append("echo_body_size", parseInt(echoBodySize));
      }
  }

  // tls
  let tlsEnabled = document.getElementById("tcpTLSEnable").checked;
  if (tlsEnabled) {
    formData.append("tls_enabled", tlsEnabled);
    // insecure
    let insecureTLS = document.getElementById("tcpTLSInsecure").checked;
    let tlsCA = document.getElementById("tcpTLSCA").files[0];
    if (insecureTLS) {
      formData.append("tls_insecure", insecureTLS);
    } else if (tlsCA) {
      formData.append("tls_ca", tlsCA);
    }
    // user certificate
    let userCert = document.getElementById("tcpTLSUserCert").files[0];
    let userKey = document.getElementById("tcpTLSUserKey").files[0];
    if (userCert && userKey){
      formData.append("tls_user_cert", userCert);
      formData.append("tls_user_key", userKey);
    }
  }

  // building request
  let xhr = new XMLHttpRequest();
  xhr.onreadystatechange = function() {
    if (this.readyState == 4) {
      if (this.status == 200) {
        let answer = escapeHTML(this.responseText.replaceAll(/[\n\r]/g, "##NEW_LINE##")).replaceAll("##NEW_LINE##", "<br />");
        resultOk(resultP, "success<br />"+answer, button);
      } else if (this.status != 0) {
        resultError(resultP, this.responseText, button);
      } else {
        resultConnectionError(resultP, button);
      }
    }
  };

  // sending request
  xhr.open("POST", "/tcp", true);
  xhr.send(formData);
}

function tcpEchoBodyChanged(checked) {
  document.getElementById("tcpEchoBodySize").disabled = !checked;
}

function tcpTLSEnabledChanged(checked) {
  showFormSection('tcpTLSConfig', checked);
}

function tcpTLSInsecureChanged(checked) {
  document.getElementById("tcpTLSCA").disabled = checked;
}

// cors
const httpURLRegexp = new RegExp("^https?://");
const websocketURLRegexp = new RegExp("^wss?://");
function cors(button, resultP) {
  clearOldResult(button, resultP);

  let method = document.getElementById("corsMethod").value;
  let url = document.getElementById("corsURL").value;

  // check method selected
  if (method.length == 0) {
    resultError(resultP, "no method selected", button);
    return;
  }
  // check URL
  if (url.length == 0) {
    resultError(resultP, "empty URL", button);
    return;
  }
  if (httpURLRegexp.test(url)) { // regular request
    // building request
    var xhr = new XMLHttpRequest();
    xhr.onreadystatechange = function() {
      if (this.readyState == 4) {
        if (this.status == 0) {
          resultConnectionError(resultP, button);
          return;
        }
        let resultText = "Server responded with status: <strong>" + this.status + "</strong><br /><br /><strong>Response headers: </strong><br />";
        let responseHeaders = xhr.getAllResponseHeaders().trim().split(/[\r\n]+/);
        responseHeaders.forEach((header) => {
          resultText += header + "<br />";
        });
        if (this.responseText.length > 0) {
          resultText += "<br /><strong>Content: </strong><br />" + escapeHTML(this.responseText);
        } else {
          resultText += "<br /><strong>No content</strong>";
        }
        if ((this.status >= 200) && (this.status < 300)) {
          resultOk(resultP, resultText, button);
        } else {
          resultError(resultP, resultText, button);
        }
      }
    };

    // sending request
    xhr.open(method, url, true);
    xhr.send();
  } else if (websocketURLRegexp.test(url)) { // websocket request
    var websocket = new WebSocket(url);
    websocket.onopen = function(event) {
      resultError(resultP, "websocket successfuly connected, closing connection", button);
      websocket.close(1000, "just testing connectivity with the Integration Test Server"); // normal closure
    };
    websocket.onerror = function(event) {
      console.log("websocket error:");
      console.log(event);
      resultError(resultP, "failed to open websocket connection, more information in the console", button);
      websocket.close(1000, "just testing connectivity with the Integration Test Server"); // normal closure
    };
  } else {
    resultError(resultP, "URL must start with \"http://\", \"https://\", \"ws://\" or \"wss://\"", button);
  }
}

// common monitoring
function monitoringGet(button, resultP, url) {
  clearOldResult(button, resultP);

  // building request
  let xhr = new XMLHttpRequest();
  xhr.onreadystatechange = function() {
    if (this.readyState == 4) {
      if ((this.status >= 200) && (this.status < 300)) {
        resultOk(resultP, "ok (status code: " + this.status + ")", button);
      } else if (this.status != 0) {
        resultError(resultP, "incorrect status code: "+ this.status + ", error: " + this.responseText, button);
      } else {
        resultConnectionError(resultP, button);
      }
    }
  };

  // sending request
  xhr.open("GET", url, true);
  xhr.send();
}

function monitoringPost(button, resultP, url) {
  clearOldResult(button, resultP);

  // inputs
  let inputIdPrefix = url.substring(1);
  let fail = document.getElementById(inputIdPrefix+"Fail").checked;
  url += "?fail=" + (fail ? "true" : "false");
  let nbFailures = document.getElementById(inputIdPrefix+"NbFailures").value;
  if (nbFailures.length > 0 ){
    if (parseInt(nbFailures) < 0) {
      resultError(resultP, "number of failures inferior to 0", button);
      return;
    }
    url += "&nb_failures=" + parseInt(nbFailures);
  }
  let delay = document.getElementById(inputIdPrefix+"Delay").value;
  if (delay.length > 0 ){
    url += "&delay=" + delay.trim();
  }

  // building request
  let xhr = new XMLHttpRequest();
  xhr.onreadystatechange = function() {
    if (this.readyState == 4) {
      if (this.status == 200) {
        resultOk(resultP, "healthcheck configured", button);
      } else if (this.status != 0) {
        resultError(resultP, this.responseText, button);
      } else {
        resultConnectionError(resultP, button);
      }
    }
  };

  // sending request
  xhr.open("POST", url, true);
  xhr.send();
}

// common
function clearOldResult(button, resultP) {
  button.disabled = true;
  resultP.style.display = "none";
  resultP.innerHTML = "";
}

function resultOk(resultP, text, button) {
  result(resultP, "text-success", text, button);
}

function resultWarning(resultP, text, button) {
  result(resultP, "text-warning", text, button);
}

function resultError(resultP, text, button) {
  result(resultP, "text-danger", "an error occured: " + text, button);
}

function resultConnectionError(resultP, button) {
  result(resultP, "text-danger", "failed to connect to the server", button);
}

function result(resultP, resultPClass, text, button) {
  resultP.className = resultPClass;
  resultP.innerHTML = text;
  resultP.style.display = "block";
  button.disabled = false;
}

function showFormSection(sectionId, enabled) {
  document.getElementById(sectionId).style.display = enabled ? "block" : "none";
}

const sizePowers = ["B", "KiB", "MiB", "GiB", "TiB", "PiB"];
function sizeToString(size, decimal = 2, power = 0) {
  size = parseFloat(size);
  if (size >= 1024) {
    return sizeToString(size / 1024, parseInt(decimal), parseInt(power) + 1);
  }
  size = Math.round(size * Math.pow(10, parseInt(decimal))) / Math.pow(10, parseInt(decimal)); // rounding with decimals
  return size + sizePowers[power];
}

function escapeHTML(s) {
  return s.replace(/&/g, "&amp;")
  .replace(/</g, "&lt;")
  .replace(/>/g, "&gt;")
  .replace(/"/g, "&quot;")
  .replace(/'/g, "&#039;");
}
