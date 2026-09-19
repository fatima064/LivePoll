import { useState , useEffect } from "react";
import "./App.css";

const API = "http://localhost:8080";
const POLL_ID = "6aae4752cb7d5ce14f48e085";

function App() {
  const [mode, setMode] = useState("login");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [token, setToken] = useState(localStorage.getItem("token"));

  const [question, setQuestion] = useState("");
  const [options, setOptions] = useState(["", ""]);
  const [message, setMessage] = useState("");
  const [poll, setPoll] = useState(null);
  const [results, setResults] = useState({});
 useEffect(() => {
  const loadPoll = async () => {
    try {
      const response = await fetch(`${API}/api/polls/${POLL_ID}`);
      const data = await response.json();

      if (response.ok) {
        setPoll(data);
      }

      const resultsResponse = await fetch(
        `${API}/api/polls/${POLL_ID}/results`
      );

      const resultsData = await resultsResponse.json();

      if (resultsResponse.ok) {
        setResults(resultsData);
      }
    } catch (error) {
      console.error("Failed to load poll:", error);
    }
  };

  loadPoll();
}, []);
  useEffect(() => {
    if (!poll) return;

    const eventSource = new EventSource(
      `${API}/api/polls/${POLL_ID}/live`
    );

    eventSource.onmessage = (event) => {
      try {
        const liveUpdate = JSON.parse(event.data);

        setResults((previousResults) => ({
          ...previousResults,
          [liveUpdate.optionId]: String(liveUpdate.count),
        }));
      } catch (error) {
        console.error("Live update error:", error);
      }
    };

    eventSource.onerror = (error) => {
      console.error("Live connection error:", error);
    };

    return () => {
      eventSource.close();
    };
  }, [poll]);

  const handleAuth = async (e) => {
    e.preventDefault();
    setMessage("");

    try {
      const response = await fetch(`${API}/api/${mode === "login" ? "login" : "signup"}`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
  email: username,
  password,
}),
 
      });

      const data = await response.json();

      if (!response.ok) {
        setMessage(data.error || "Something went wrong.");
        return;
      }

      if (mode === "signup") {
        setMessage("Signup successful! Now login.");
        setMode("login");
        return;
      }

      localStorage.setItem("token", data.token);
      setToken(data.token);
      setMessage("Login successful! 🎉");
    } catch {
      setMessage("Cannot connect to backend.");
    }
  };

  const addOption = () => {
    if (options.length < 6) {
      setOptions([...options, ""]);
    }
  };

  const updateOption = (index, value) => {
    const updated = [...options];
    updated[index] = value;
    setOptions(updated);
  };
const vote = async (optionId) => {
  try {
    const response = await fetch(
      `${API}/api/polls/${POLL_ID}/vote`,
      {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          optionId: optionId,
        }),
      }
    );

    const data = await response.json();

   if (!response.ok) {
  alert(data.error || "Vote failed");
  return;
}

const resultsResponse = await fetch(
  `${API}/api/polls/${POLL_ID}/results`
);

const resultsData = await resultsResponse.json();

if (resultsResponse.ok) {
  setResults(resultsData);
}

alert("Vote recorded! 🎉");
  } catch (error) {
    console.error(error);
    alert("Could not connect to backend.");
  }
};
  const createPoll = async (e) => {
    e.preventDefault();
    setMessage("");

    const cleanOptions = options.filter((x) => x.trim());

    if (!question.trim()) {
      setMessage("Please enter a question.");
      return;
    }

    if (cleanOptions.length < 2) {
      setMessage("Please add at least 2 options.");
      return;
    }

    try {
      const response = await fetch(`${API}/api/polls`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({
          question,
          options: cleanOptions,
        }),
      });

      const data = await response.json();

      if (!response.ok) {
        setMessage(data.error || "Could not create poll.");
        return;
      }

      setPoll(data);
      setMessage("Poll created successfully! 🎉");
      setQuestion("");
      setOptions(["", ""]);
    } catch {
      setMessage("Cannot connect to backend.");
    }
  };

  const logout = () => {
    localStorage.removeItem("token");
    setToken(null);
    setPoll(null);
    setMessage("");
  };

  if (!token) {
    return (
      <div className="app">
        <header className="header">
          <h1>LivePoll</h1>
          <p>Real-time polling made simple.</p>
        </header>

        <main className="container">
          <section className="card">
            <h2>{mode === "login" ? "Login" : "Create Account"}</h2>

            <form onSubmit={handleAuth}>
              <label>Username</label>

              <input
                type="text"
                placeholder="Enter username"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                required
              />

              <label>Password</label>

              <input
                type="password"
                placeholder="Enter password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
              />

              <button className="primary">
                {mode === "login" ? "Login" : "Sign Up"}
              </button>
            </form>

            {message && <p className="message">{message}</p>}

            <button
              className="secondary"
              onClick={() => {
                setMode(mode === "login" ? "signup" : "login");
                setMessage("");
              }}
            >
              {mode === "login"
                ? "Create a new account"
                : "Already have an account? Login"}
            </button>
          </section>
        </main>
      </div>
    );
  }

  return (
    <div className="app">
      <header className="header">
        <div>
          <h1>LivePoll</h1>
          <p>Create polls and see results update live.</p>
        </div>

        <button className="logout" onClick={logout}>
          Logout
        </button>
      </header>

      <main className="container">
        <section className="card">
          <h2>Create a Poll</h2>

          <form onSubmit={createPoll}>
            <label>Question</label>

            <input
              type="text"
              placeholder="What should we have for lunch?"
              value={question}
              onChange={(e) => setQuestion(e.target.value)}
            />

            <label>Options</label>

            {options.map((option, index) => (
              <input
                key={index}
                type="text"
                placeholder={`Option ${index + 1}`}
                value={option}
                onChange={(e) => updateOption(index, e.target.value)}
              />
            ))}

            {options.length < 6 && (
              <button type="button" className="secondary" onClick={addOption}>
                + Add Option
              </button>
            )}

            <button type="submit" className="primary">
              Create Poll
            </button>
          </form>

          {message && <p className="message">{message}</p>}
        </section>

        {poll && (
          <section className="card">
            <h2>Poll Created 🎉</h2>

            <h3>{poll.question}</h3>

            {poll.options?.map((option, index) => (
  <button
    className="poll-option"
    key={index}
    onClick={() => vote(option.id)}
  >
    {option.text} — {results[`option-${index + 1}`] || 0} votes
  </button>
))}

            <p className="poll-id">
              Poll ID: {poll.id || poll._id}
            </p>
          </section>
        )}
      </main>
    </div>
  );
}

export default App;