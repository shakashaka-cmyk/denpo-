import React, { useState, useEffect, } from 'react';
import './DenpouApp.css';

const API_BASE = 'http://localhost:8080/api';

const DenpouApp = () => {
  const [appState, setAppState] = useState('setup'); // setup, game, result
  const [playerNames, setPlayerNames] = useState(['', '', '', '']);
  const [game, setGame] = useState(null);
  const [currentRound, setCurrentRound] = useState(0);
  const [gamePhase, setGamePhase] = useState('waiting'); // waiting, hint_phase, answering, finished
  const [answer, setAnswer] = useState('');
  const [hintText, setHintText] = useState('');
  const [hints, setHints] = useState([]);
  const [currentPlayerID, setCurrentPlayerID] = useState(null);
  const [errorMsg, setErrorMsg] = useState('');

  // ゲーム作成
  const createGame = async () => {
    if (playerNames.filter(n => n.trim()).length < 2) {
      setErrorMsg('最低2人のプレイヤーが必要です');
      return;
    }

    const playersData = playerNames.filter(n => n.trim()).map(name => ({ name }));
    
    try {
      const response = await fetch(`${API_BASE}/games`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ players: playersData }),
      });
      const gameData = await response.json();
      setGame(gameData);
      setCurrentPlayerID(gameData.players[0].playerId);
      setAppState('game');
      setErrorMsg('');
    } catch (err) {
      setErrorMsg('ゲーム作成に失敗しました');
    }
  };

  // ゲーム状態を取得
  useEffect(() => {
    if (game && appState === 'game') {
      const interval = setInterval(async () => {
        try {
          const response = await fetch(`${API_BASE}/games/${game.gameId}`);
          const updated = await response.json();
          setGame(updated);
        } catch (err) {
          console.error('Failed to fetch game:', err);
        }
      }, 1000);
      return () => clearInterval(interval);
    }
  }, [game, appState]);

  // ラウンドを開始
  const startRound = async () => {
    if (!answer.trim()) {
      setErrorMsg('答えを入力してください');
      return;
    }

    try {
      const response = await fetch(
        `${API_BASE}/games/${game.gameId}/rounds/${currentRound + 1}/start`,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'X-Player-ID': currentPlayerID,
          },
          body: JSON.stringify({ answer }),
        }
      );
      setGamePhase('hint_phase');
      setAnswer('');
      setHints([]);
      setErrorMsg('');
    } catch (err) {
      setErrorMsg('ラウンド開始に失敗しました');
    }
  };

  // ヒントを投稿
  const submitHint = async () => {
    if (!hintText.trim()) {
      setErrorMsg('ヒントを入力してください');
      return;
    }

    try {
      const response = await fetch(
        `${API_BASE}/games/${game.gameId}/rounds/${currentRound + 1}/hints`,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'X-Player-ID': currentPlayerID,
          },
          body: JSON.stringify({ text: hintText }),
        }
      );
      const hint = await response.json();
      setHints([...hints, hint]);
      setHintText('');
      setErrorMsg('');
    } catch (err) {
      setErrorMsg('ヒント投稿に失敗しました');
    }
  };

  // 解答を投稿
  const submitAnswer = async () => {
    if (!answer.trim()) {
      setErrorMsg('答えを入力してください');
      return;
    }

    try {
      const response = await fetch(
        `${API_BASE}/games/${game.gameId}/rounds/${currentRound + 1}/answer`,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'X-Player-ID': currentPlayerID,
          },
          body: JSON.stringify({ answer }),
        }
      );
      setGamePhase('finished');
      setAnswer('');
    } catch (err) {
      setErrorMsg('解答送信に失敗しました');
    }
  };

  // セットアップ画面
  if (appState === 'setup') {
    return (
      <div className="denpou-container setup-screen">
        <div className="setup-header">
          <h1 className="title">デンポー！！</h1>
          <p className="subtitle">文字数制限でヒントを出す、創意工夫のゲーム</p>
        </div>

        <div className="setup-form">
          <h2>プレイヤーを入力</h2>
          <div className="player-inputs">
            {playerNames.map((name, idx) => (
              <div key={idx} className="player-input-group">
                <label>プレイヤー {idx + 1}</label>
                <input
                  type="text"
                  value={name}
                  onChange={(e) => {
                    const newNames = [...playerNames];
                    newNames[idx] = e.target.value;
                    setPlayerNames(newNames);
                  }}
                  placeholder={`プレイヤー${idx + 1}の名前`}
                />
              </div>
            ))}
          </div>

          {errorMsg && <div className="error-message">{errorMsg}</div>}

          <button className="btn-primary" onClick={createGame}>
            ゲームを開始
          </button>
        </div>

        <div className="rules-section">
          <h3>ルール</h3>
          <ul>
            <li>親が答えを決める</li>
            <li>子が少ない文字数でヒントを出す</li>
            <li>スコア = 18 - (文字数 ÷ 順番)※小数点切り上げ</li>
            <li>正解で親と解答者に同じ得点</li>
            <li>親を2周、合計得点を競う</li>
          </ul>
        </div>
      </div>
    );
  }

  if (!game) return <div>読み込み中...</div>;

  const currentRoundData = game.rounds[currentRound];
  const isParent = currentPlayerID === currentRoundData?.parentId;

  // ゲーム画面
  if (appState === 'game') {
    return (
      <div className="denpou-container game-screen">
        <header className="game-header">
          <h1 className="title">デンポー！！</h1>
          <div className="round-info">
            ラウンド {currentRound + 1} / {game.rounds.length}
          </div>
        </header>

        <div className="game-board">
          {/* スコアボード */}
          <div className="scoreboard">
            <h3>スコア</h3>
            <div className="scores">
              {game.players.map((player) => (
                <div key={player.playerId} className="score-item">
                  <span className="player-name">{player.name}</span>
                  <span className="player-score">{player.totalScore}</span>
                </div>
              ))}
            </div>
          </div>

          {/* フェーズ1: 親が答えを決定 */}
          {isParent && gamePhase === 'waiting' && (
            <div className="phase-box parent-setup">
              <h2>親のターン</h2>
              <p>ヒントの対象となる答えを入力してください</p>
              <input
                type="text"
                value={answer}
                onChange={(e) => setAnswer(e.target.value)}
                placeholder="例：ピラミッド"
                className="answer-input"
              />
              <button className="btn-primary" onClick={startRound}>
                答えを決定 → ヒント受付開始
              </button>
            </div>
          )}

          {/* フェーズ2: 子がヒントを出す */}
          {!isParent && gamePhase === 'hint_phase' && (
            <div className="phase-box hint-submission">
              <h2>ヒント出題フェーズ</h2>
              <p>
                親は <strong>"{currentRoundData?.answer}"</strong> です。
              </p>
              <p className="hint-instruction">
                これを当てるヒントを出してください（文字数が少ないほど高得点！）
              </p>

              <div className="hint-input-area">
                <textarea
                  value={hintText}
                  onChange={(e) => setHintText(e.target.value)}
                  placeholder="ヒントを入力..."
                  className="hint-textarea"
                />
                <div className="hint-stats">
                  <span className="char-count">
                    文字数: {hintText.length}
                  </span>
                  {hintText.length > 0 && (
                    <span className="potential-score">
                      予想スコア: {18 - Math.ceil(hintText.length / (hints.length + 1))}
                    </span>
                  )}
                </div>
                <button
                  className="btn-secondary"
                  onClick={submitHint}
                  disabled={!hintText.trim()}
                >
                  ヒントを投稿
                </button>
              </div>

              {/* ヒント一覧 */}
              {hints.length > 0 && (
                <div className="hints-display">
                  <h4>投稿されたヒント</h4>
                  <div className="hints-list">
                    {hints.map((hint, idx) => (
                      <div key={idx} className="hint-item">
                        <span className="hint-order">#{hint.order}</span>
                        <span className="hint-text">{hint.text}</span>
                        <span className="hint-meta">
                          {hint.charCount}字 → {hint.score}点
                        </span>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          )}

          {/* フェーズ3: 解答タイム */}
          {isParent && gamePhase === 'hint_phase' && (
            <div className="phase-box answer-waiting">
              <h2>ヒント出題中...</h2>
              <p>子たちのヒントを待っています</p>
              <div className="waiting-animation">
                <div className="spinner"></div>
              </div>
              {hints.length > 0 && (
                <div className="current-hints">
                  <h4>現在のヒント:</h4>
                  <div className="hints-display">
                    {hints.map((hint, idx) => (
                      <div key={idx} className="hint-card">
                        <div className="hint-text-display">{hint.text}</div>
                        <div className="hint-char-count">{hint.charCount}字</div>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          )}

          {/* フェーズ4: 解答フェーズ */}
          {gamePhase === 'answering' && (
            <div className="phase-box answer-phase">
              <h2>解答フェーズ 30秒</h2>
              <div className="all-hints">
                <h4>すべてのヒント:</h4>
                {hints.map((hint, idx) => (
                  <div key={idx} className="hint-card-large">
                    <div className="hint-number">{hint.order}</div>
                    <div className="hint-text-large">{hint.text}</div>
                    <div className="hint-info">
                      {hint.charCount}字 • {hint.score}点
                    </div>
                  </div>
                ))}
              </div>

              <div className="answer-input-area">
                <input
                  type="text"
                  value={answer}
                  onChange={(e) => setAnswer(e.target.value)}
                  placeholder="答えは何？"
                  className="answer-input-large"
                  autoFocus
                />
                <button className="btn-primary-large" onClick={submitAnswer}>
                  答える
                </button>
              </div>
            </div>
          )}

          {/* フェーズ5: ラウンド終了 */}
          {gamePhase === 'finished' && (
            <div className="phase-box round-result">
              <h2>ラウンド終了</h2>
              <div className="result-content">
                <p>正解: {currentRoundData?.answer}</p>
                <p>
                  {currentRoundData?.correctAnswer
                    ? '✅ 正解！'
                    : '❌ ハズレ'}
                </p>
              </div>

              {currentRound < game.rounds.length - 1 ? (
                <button
                  className="btn-primary"
                  onClick={() => {
                    setCurrentRound(currentRound + 1);
                    setGamePhase('waiting');
                    setHints([]);
                    setAnswer('');
                    setHintText('');
                  }}
                >
                  次のラウンドへ
                </button>
              ) : (
                <button
                  className="btn-primary"
                  onClick={() => setAppState('result')}
                >
                  最終結果を見る
                </button>
              )}
            </div>
          )}

          {errorMsg && <div className="error-message">{errorMsg}</div>}
        </div>
      </div>
    );
  }

  // 結果画面
  if (appState === 'result') {
    const sorted = [...game.players].sort((a, b) => b.totalScore - a.totalScore);
    return (
      <div className="denpou-container result-screen">
        <h1 className="title">ゲーム終了！</h1>
        <div className="final-ranking">
          <h2>最終順位</h2>
          <div className="ranking-list">
            {sorted.map((player, idx) => (
              <div key={player.playerId} className="ranking-item">
                <span className="ranking-medal">
                  {idx === 0 && '🥇'}
                  {idx === 1 && '🥈'}
                  {idx === 2 && '🥉'}
                  {idx >= 3 && `${idx + 1}位`}
                </span>
                <span className="ranking-name">{player.name}</span>
                <span className="ranking-score">{player.totalScore}点</span>
              </div>
            ))}
          </div>
        </div>
        <button
          className="btn-primary"
          onClick={() => {
            setAppState('setup');
            setPlayerNames(['', '', '', '']);
            setGame(null);
          }}
        >
          新しいゲームを開始
        </button>
      </div>
    );
  }
};

export default DenpouApp;
