import React, { useState, useEffect } from 'react';

const API_BASE = process.env.REACT_APP_API_BASE || 'http://localhost:8080/api';

const DenpouApp = () => {
  const [appState, setAppState] = useState('lobby');
  const [gameID, setGameID] = useState(null);
  const [playerID, setPlayerID] = useState(null);
  const [playerNames, setPlayerNames] = useState(['', '', '', '']);
  const [game, setGame] = useState(null);
  const [currentRound, setCurrentRound] = useState(0);
  const [answer, setAnswer] = useState('');
  const [hintText, setHintText] = useState('');
  const [errorMsg, setErrorMsg] = useState('');
  const [ws, setWs] = useState(null);

  useEffect(() => {
    if (gameID && playerID) {
      const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      const hostPart = API_BASE.split('//')[1].split('/')[0];
      const wsURL = `${wsProtocol}//${hostPart}/api/games/${gameID}/ws`;
      
      const websocket = new WebSocket(wsURL);
      websocket.onmessage = (event) => {
        const updatedGame = JSON.parse(event.data);
        setGame(updatedGame);
      };
      setWs(websocket);

      return () => {
        if (websocket && websocket.readyState === WebSocket.OPEN) {
          websocket.close();
        }
      };
    }
  }, [gameID, playerID]);

  const createGame = async () => {
    const players = playerNames.filter(n => n.trim()).map(name => ({ name }));
    
    if (players.length < 2) {
      setErrorMsg('最低2人のプレイヤーが必要です');
      return;
    }

    try {
      const response = await fetch(`${API_BASE}/games`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ players }),
      });
      const gameData = await response.json();
      
      setGameID(gameData.gameId);
      setPlayerID(gameData.players[0].playerId);
      setAppState('game');
    } catch (err) {
      setErrorMsg('ゲーム作成に失敗しました');
    }
  };

  const startRound = async () => {
    if (!answer.trim()) return;
    try {
      await fetch(`${API_BASE}/games/${gameID}/rounds/start`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'X-Player-ID': playerID },
        body: JSON.stringify({ answer }),
      });
      setAnswer('');
    } catch (err) {
      setErrorMsg('失敗しました');
    }
  };

  const submitHint = async () => {
    if (!hintText.trim()) return;
    try {
      await fetch(`${API_BASE}/games/${gameID}/hints`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'X-Player-ID': playerID },
        body: JSON.stringify({ text: hintText }),
      });
      setHintText('');
    } catch (err) {
      setErrorMsg('失敗しました');
    }
  };

  if (!game && appState === 'game') return <div>読み込み中...</div>;

  if (appState === 'lobby') {
    return (
      <div style={{ maxWidth: '600px', margin: '0 auto', padding: '20px' }}>
        <h1 style={{ textAlign: 'center', fontSize: '2rem', color: '#E74C3C' }}>デンポー！！</h1>
        <div style={{ background: '#fff', padding: '20px', borderRadius: '8px' }}>
          <h2>プレイヤーを入力</h2>
          {playerNames.map((name, idx) => (
            <div key={idx} style={{ marginBottom: '10px' }}>
              <input
                type="text"
                value={name}
                onChange={(e) => {
                  const newNames = [...playerNames];
                  newNames[idx] = e.target.value;
                  setPlayerNames(newNames);
                }}
                placeholder={`プレイヤー${idx + 1}`}
                style={{ width: '100%', padding: '8px', marginBottom: '5px' }}
              />
            </div>
          ))}
          <button onClick={createGame} style={{ width: '100%', padding: '10px', background: '#E74C3C', color: '#fff', border: 'none', cursor: 'pointer' }}>
            ゲーム開始
          </button>
          {errorMsg && <div style={{ color: '#E74C3C', marginTop: '10px' }}>{errorMsg}</div>}
        </div>
      </div>
    );
  }

  if (appState === 'game' && game) {
    const currentRoundData = game.rounds[currentRound];
    const isParent = playerID === currentRoundData?.parentId;

    return (
      <div style={{ maxWidth: '900px', margin: '0 auto', padding: '20px' }}>
        <h1 style={{ textAlign: 'center' }}>デンポー！！</h1>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 2fr', gap: '20px' }}>
          <div style={{ background: '#fff', padding: '20px', borderRadius: '8px' }}>
            <h3>スコア</h3>
            {game.players.map((p) => (
              <div key={p.playerId} style={{ padding: '10px', background: '#f5f5f5', marginBottom: '10px' }}>
                <div style={{ fontWeight: 'bold' }}>{p.name} {p.playerId === playerID && '(あなた)'}</div>
                <div style={{ fontSize: '1.2rem', color: '#E74C3C' }}>{p.totalScore}点</div>
              </div>
            ))}
          </div>

          <div style={{ background: '#fff', padding: '20px', borderRadius: '8px' }}>
            {isParent && currentRoundData?.status === 'waiting' && (
              <div>
                <h2>親のターン</h2>
                <input type="text" value={answer} onChange={(e) => setAnswer(e.target.value)} placeholder="答えを入力" style={{ width: '100%', padding: '8px', marginBottom: '10px' }} />
                <button onClick={startRound} style={{ width: '100%', padding: '10px', background: '#E74C3C', color: '#fff', border: 'none', cursor: 'pointer' }}>答えを決定</button>
              </div>
            )}

            {!isParent && currentRoundData?.status === 'hint_phase' && (
              <div>
                <h2>ヒント出題</h2>
                <p>親は「{currentRoundData?.answer}」です</p>
                <textarea value={hintText} onChange={(e) => setHintText(e.target.value)} placeholder="ヒントを入力" style={{ width: '100%', minHeight: '80px', padding: '8px', marginBottom: '10px' }} />
                <div style={{ marginBottom: '10px' }}>文字数: {hintText.length}</div>
                <button onClick={submitHint} style={{ width: '100%', padding: '10px', background: '#F39C12', color: '#fff', border: 'none', cursor: 'pointer' }}>投稿</button>

                {currentRoundData.hints.length > 0 && (
                  <div style={{ marginTop: '20px', paddingTop: '20px', borderTop: '1px solid #ddd' }}>
                    <h4>ヒント一覧</h4>
                    {currentRoundData.hints.map((hint, idx) => (
                      <div key={idx} style={{ background: '#f5f5f5', padding: '10px', marginBottom: '5px' }}>
                        {hint.text} ({hint.charCount}字)
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}

            {isParent && currentRoundData?.status === 'hint_phase' && (
              <div>
                <h2>ヒント出題中...</h2>
                {currentRoundData.hints.map((hint, idx) => (
                  <div key={idx} style={{ background: '#f5f5f5', padding: '10px', marginBottom: '10px' }}>
                    {hint.text} ({hint.charCount}字)
                  </div>
                ))}
              </div>
            )}

            {currentRoundData?.status === 'finished' && (
              <div>
                <h2>ラウンド終了</h2>
                <p>正解: {currentRoundData?.answer}</p>
                <p>{currentRoundData?.correctAnswer ? '✅ 正解！' : '❌ ハズレ'}</p>
                {currentRound < game.rounds.length - 1 ? (
                  <button onClick={() => { setCurrentRound(currentRound + 1); setAnswer(''); setHintText(''); }} style={{ width: '100%', padding: '10px', background: '#E74C3C', color: '#fff', border: 'none', cursor: 'pointer' }}>次へ</button>
                ) : (
                  <button onClick={() => setAppState('result')} style={{ width: '100%', padding: '10px', background: '#E74C3C', color: '#fff', border: 'none', cursor: 'pointer' }}>結果を見る</button>
                )}
              </div>
            )}
          </div>
        </div>
      </div>
    );
  }

  if (appState === 'result' && game) {
    const sorted = [...game.players].sort((a, b) => b.totalScore - a.totalScore);
    return (
      <div style={{ maxWidth: '600px', margin: '0 auto', padding: '20px' }}>
        <h1 style={{ textAlign: 'center' }}>ゲーム終了！</h1>
        {sorted.map((p, idx) => (
          <div key={p.playerId} style={{ background: '#f5f5f5', padding: '15px', marginBottom: '10px', borderRadius: '8px' }}>
            <span>{idx === 0 ? '🥇' : idx === 1 ? '🥈' : idx === 2 ? '🥉' : `${idx + 1}位`} {p.name}: {p.totalScore}点</span>
          </div>
        ))}
        <button onClick={() => { setAppState('lobby'); setGameID(null); setPlayerID(null); setGame(null); setPlayerNames(['', '', '', '']); }} style={{ width: '100%', padding: '10px', background: '#E74C3C', color: '#fff', border: 'none', cursor: 'pointer', marginTop: '20px' }}>新規ゲーム</button>
      </div>
    );
  }

  return <div>読み込み中...</div>;
};

export default DenpouApp;