import React, { useState, useEffect } from 'react';

const API_BASE = process.env.REACT_APP_API_BASE || 'http://localhost:8080/api';

const DenpouApp = () => {
  const [appState, setAppState] = useState('lobby');
  const [gameID, setGameID] = useState(null);
  const [playerID, setPlayerID] = useState(null);
  const [parentName, setParentName] = useState('');
  const [game, setGame] = useState(null);
  const [currentRound, setCurrentRound] = useState(0);
  const [answer, setAnswer] = useState('');
  const [hintText, setHintText] = useState('');
  const [errorMsg, setErrorMsg] = useState('');
  const [, setWs] = useState(null);
  const [joinGameID, setJoinGameID] = useState('');
  const [playerName, setPlayerName] = useState('');

  // WebSocket接続
  useEffect(() => {
    if (gameID && playerID) {
      const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      const hostPart = API_BASE.split('//')[1].split('/')[0];
      const wsURL = `${wsProtocol}//${hostPart}/api/games/${gameID}/ws`;
      
      try {
        const websocket = new WebSocket(wsURL);

        websocket.onmessage = (event) => {
          try {
            const updatedGame = JSON.parse(event.data);
            setGame(updatedGame);
          } catch (e) {
            console.error('WebSocket parse error:', e);
          }
        };

        websocket.onerror = (error) => {
          console.error('WebSocket error:', error);
        };

        setWs(websocket);

        return () => {
          if (websocket && websocket.readyState === WebSocket.OPEN) {
            websocket.close();
          }
        };
      } catch (e) {
        console.error('WebSocket connection error:', e);
      }
    }
  }, [gameID, playerID]);

  // ゲーム作成（親だけ）
  const createGame = async () => {
    if (!parentName.trim()) {
      setErrorMsg('名前を入力してください');
      return;
    }

    try {
      const response = await fetch(`${API_BASE}/games`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ players: [{ name: parentName }] }),
      });
      const gameData = await response.json();
      
      setGameID(gameData.gameId);
      setPlayerID(gameData.players[0].playerId);
      setAppState('game');
      setErrorMsg('');
    } catch (err) {
      setErrorMsg('ゲーム作成に失敗しました');
      console.error(err);
    }
  };

  // ゲームに参加
  const joinGame = async () => {
    if (!joinGameID.trim()) {
      setErrorMsg('部屋IDを入力してください');
      return;
    }

    if (!playerName.trim()) {
      setErrorMsg('プレイヤー名を入力してください');
      return;
    }

    try {
      const response = await fetch(`${API_BASE}/games/${joinGameID}`, {
        method: 'GET',
      });
      
      if (!response.ok) {
        setErrorMsg('その部屋は見つかりません');
        return;
      }

      const gameData = await response.json();
      
      // ランダムにプレイヤーを選択
      const randomIndex = Math.floor(Math.random() * gameData.players.length);
      const selectedPlayerID = gameData.players[randomIndex].playerId;
      
      setGameID(joinGameID);
      setPlayerID(selectedPlayerID);
      setAppState('game');
      setErrorMsg('');
    } catch (err) {
      setErrorMsg('参加に失敗しました');
      console.error(err);
    }
  };

  // ラウンド開始
  const startRound = async () => {
    if (!answer.trim()) {
      setErrorMsg('答えを入力してください');
      return;
    }

    try {
      await fetch(`${API_BASE}/games/${gameID}/rounds/start`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Player-ID': playerID,
        },
        body: JSON.stringify({ answer }),
      });
      setAnswer('');
      setErrorMsg('');
    } catch (err) {
      setErrorMsg('失敗しました');
      console.error(err);
    }
  };

  // ヒント投稿
  const submitHint = async () => {
    if (!hintText.trim()) {
      setErrorMsg('ヒントを入力してください');
      return;
    }

    try {
      await fetch(`${API_BASE}/games/${gameID}/hints`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Player-ID': playerID,
        },
        body: JSON.stringify({ text: hintText }),
      });
      setHintText('');
      setErrorMsg('');
    } catch (err) {
      setErrorMsg('失敗しました');
      console.error(err);
    }
  };

  // ロビー画面
  if (appState === 'lobby') {
    return (
      <div style={{ maxWidth: '900px', margin: '0 auto', padding: '20px' }}>
        <h1 style={{ textAlign: 'center', fontSize: '2.5rem', color: '#E74C3C', marginBottom: '10px' }}>
          デンポー！！
        </h1>
        <p style={{ textAlign: 'center', color: '#666', marginBottom: '40px' }}>
          文字数制限でヒントを出す、創意工夫のゲーム
        </p>

        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '20px', marginBottom: '40px' }}>
          {/* 部屋作成 */}
          <div style={{ background: '#fff', padding: '30px', borderRadius: '12px', boxShadow: '0 5px 20px rgba(0,0,0,0.1)' }}>
            <h2 style={{ marginBottom: '20px', color: '#2C3E50' }}>新しい部屋を作成</h2>
            
            <div style={{ marginBottom: '15px' }}>
              <label style={{ display: 'block', marginBottom: '5px', fontWeight: 'bold', color: '#2C3E50' }}>
                あなたの名前（親）
              </label>
              <input
                type="text"
                value={parentName}
                onChange={(e) => setParentName(e.target.value)}
                placeholder="親の名前"
                style={{
                  width: '100%',
                  padding: '10px',
                  border: '2px solid #ddd',
                  borderRadius: '6px',
                  fontSize: '1rem',
                  fontFamily: 'inherit',
                }}
              />
            </div>

            <button
              onClick={createGame}
              style={{
                width: '100%',
                padding: '12px',
                background: 'linear-gradient(135deg, #E74C3C 0%, #C0392B 100%)',
                color: '#fff',
                border: 'none',
                borderRadius: '6px',
                fontSize: '1.1rem',
                fontWeight: 'bold',
                cursor: 'pointer',
                transition: 'all 0.3s ease',
              }}
              onMouseEnter={(e) => e.target.style.transform = 'translateY(-2px)'}
              onMouseLeave={(e) => e.target.style.transform = 'translateY(0)'}
            >
              ゲームを作成
            </button>
          </div>

          {/* 部屋参加 */}
          <div style={{ background: '#fff', padding: '30px', borderRadius: '12px', boxShadow: '0 5px 20px rgba(0,0,0,0.1)' }}>
            <h2 style={{ marginBottom: '20px', color: '#2C3E50' }}>既存の部屋に参加</h2>
            
            <div style={{ marginBottom: '15px' }}>
              <label style={{ display: 'block', marginBottom: '5px', fontWeight: 'bold', color: '#2C3E50' }}>
                あなたの名前
              </label>
              <input
                type="text"
                value={playerName}
                onChange={(e) => setPlayerName(e.target.value)}
                placeholder="プレイヤー名"
                style={{
                  width: '100%',
                  padding: '10px',
                  border: '2px solid #ddd',
                  borderRadius: '6px',
                  fontSize: '1rem',
                  fontFamily: 'inherit',
                }}
              />
            </div>

            <div style={{ marginBottom: '15px' }}>
              <label style={{ display: 'block', marginBottom: '5px', fontWeight: 'bold', color: '#2C3E50' }}>
                部屋ID
              </label>
              <input
                type="text"
                value={joinGameID}
                onChange={(e) => setJoinGameID(e.target.value)}
                placeholder="部屋IDを入力"
                style={{
                  width: '100%',
                  padding: '10px',
                  border: '2px solid #ddd',
                  borderRadius: '6px',
                  fontSize: '1rem',
                  fontFamily: 'inherit',
                }}
              />
            </div>

            <button
              onClick={joinGame}
              style={{
                width: '100%',
                padding: '12px',
                background: 'linear-gradient(135deg, #F39C12 0%, #E67E22 100%)',
                color: '#fff',
                border: 'none',
                borderRadius: '6px',
                fontSize: '1.1rem',
                fontWeight: 'bold',
                cursor: 'pointer',
                transition: 'all 0.3s ease',
              }}
              onMouseEnter={(e) => e.target.style.transform = 'translateY(-2px)'}
              onMouseLeave={(e) => e.target.style.transform = 'translateY(0)'}
            >
              参加する
            </button>
          </div>
        </div>

        {errorMsg && (
          <div style={{
            background: 'rgba(231, 76, 60, 0.1)',
            border: '2px solid #E74C3C',
            color: '#E74C3C',
            padding: '15px',
            borderRadius: '6px',
            marginBottom: '20px',
            fontWeight: 'bold',
            textAlign: 'center',
          }}>
            {errorMsg}
          </div>
        )}

        <div style={{ background: '#fff', padding: '30px', borderRadius: '12px', boxShadow: '0 5px 20px rgba(0,0,0,0.1)' }}>
          <h3 style={{ color: '#E74C3C', marginBottom: '15px' }}>ルール</h3>
          <ul style={{ listStyle: 'none', color: '#2C3E50', lineHeight: '1.8' }}>
            <li>→ 親が答えを決める</li>
            <li>→ 子が少ない文字数でヒントを出す</li>
            <li>→ スコア = 18 - (文字数 ÷ 順番)※小数点切り上げ</li>
            <li>→ 正解で親と解答者に同じ得点</li>
            <li>→ 親を2周、合計得点を競う</li>
          </ul>
        </div>
      </div>
    );
  }

  // ゲーム画面
  if (appState === 'game' && game) {
    const currentRoundData = game.rounds[currentRound];
    const isParent = playerID === currentRoundData?.parentId;

    return (
      <div style={{ maxWidth: '900px', margin: '0 auto', padding: '20px' }}>
        <div style={{ background: '#fff', padding: '20px', borderRadius: '12px', marginBottom: '20px', boxShadow: '0 5px 20px rgba(0,0,0,0.08)' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <h1 style={{ fontSize: '2rem', color: '#E74C3C', margin: 0 }}>デンポー！！</h1>
            <div>
              <div style={{ fontSize: '1rem', fontWeight: 'bold', color: '#2C3E50', marginBottom: '5px' }}>
                部屋ID: <code style={{ background: '#f5f5f5', padding: '5px 10px', borderRadius: '4px' }}>{gameID.slice(0, 8)}</code>
                <button
                  onClick={() => {
                    navigator.clipboard.writeText(gameID);
                    alert('部屋IDをコピーしました！');
                  }}
                  style={{
                    marginLeft: '10px',
                    padding: '5px 10px',
                    background: '#F39C12',
                    color: '#fff',
                    border: 'none',
                    borderRadius: '4px',
                    cursor: 'pointer',
                    fontSize: '0.9rem',
                  }}
                >
                  コピー
                </button>
              </div>
              <div style={{ fontSize: '1.1rem', fontWeight: 'bold', color: '#F39C12' }}>
                ラウンド {currentRound + 1} / {game.rounds.length}
              </div>
            </div>
          </div>
        </div>

        <div style={{ display: 'grid', gridTemplateColumns: '1fr 2fr', gap: '20px' }}>
          {/* スコアボード */}
          <div style={{ background: '#fff', padding: '20px', borderRadius: '12px', boxShadow: '0 5px 20px rgba(0,0,0,0.08)', height: 'fit-content', position: 'sticky', top: '20px' }}>
            <h3 style={{ marginBottom: '15px', borderBottom: '2px solid #E74C3C', paddingBottom: '10px', color: '#2C3E50' }}>スコア</h3>
            {game.players.map((player) => (
              <div
                key={player.playerId}
                style={{
                  display: 'flex',
                  justifyContent: 'space-between',
                  padding: '12px',
                  background: player.playerId === playerID ? '#FFF3CD' : '#f5f5f5',
                  marginBottom: '10px',
                  borderRadius: '6px',
                  border: player.playerId === playerID ? '2px solid #F39C12' : 'none',
                }}
              >
                <span style={{ fontWeight: 'bold', color: '#2C3E50' }}>
                  {player.name}
                  {player.playerId === playerID && ' (あなた)'}
                </span>
                <span style={{ fontSize: '1.2rem', fontWeight: 'bold', color: '#E74C3C' }}>
                  {player.totalScore}点
                </span>
              </div>
            ))}
          </div>

          {/* ゲーム画面 */}
          <div style={{ background: '#fff', padding: '20px', borderRadius: '12px', boxShadow: '0 5px 20px rgba(0,0,0,0.08)' }}>
            {isParent && currentRoundData?.status === 'waiting' && (
              <div>
                <h2 style={{ color: '#E74C3C', marginBottom: '15px' }}>👑 親のターン</h2>
                <p style={{ color: '#666', marginBottom: '15px' }}>ヒントの対象となる答えを入力してください</p>
                <input
                  type="text"
                  value={answer}
                  onChange={(e) => setAnswer(e.target.value)}
                  placeholder="例：ピラミッド"
                  style={{
                    width: '100%',
                    padding: '12px',
                    marginBottom: '15px',
                    border: '2px solid #ddd',
                    borderRadius: '6px',
                    fontSize: '1rem',
                    fontFamily: 'inherit',
                  }}
                />
                <button
                  onClick={startRound}
                  style={{
                    width: '100%',
                    padding: '12px',
                    background: '#E74C3C',
                    color: '#fff',
                    border: 'none',
                    borderRadius: '6px',
                    cursor: 'pointer',
                    fontWeight: 'bold',
                    fontSize: '1rem',
                  }}
                >
                  答えを決定
                </button>
              </div>
            )}

            {!isParent && currentRoundData?.status === 'hint_phase' && (
              <div>
                <h2 style={{ color: '#E74C3C', marginBottom: '15px' }}>💡 ヒント出題フェーズ</h2>
                <p style={{ color: '#666', marginBottom: '10px' }}>
                  親は <strong>「{currentRoundData?.answer}」</strong> です。
                </p>
                <p style={{ color: '#F39C12', fontWeight: 'bold', marginBottom: '15px' }}>
                  これを当てるヒントを出してください（文字数が少ないほど高得点！）
                </p>
                <textarea
                  value={hintText}
                  onChange={(e) => setHintText(e.target.value)}
                  placeholder="ヒントを入力..."
                  style={{
                    width: '100%',
                    minHeight: '80px',
                    padding: '12px',
                    marginBottom: '10px',
                    border: '2px solid #ddd',
                    borderRadius: '6px',
                    fontSize: '1rem',
                    fontFamily: 'inherit',
                    resize: 'vertical',
                  }}
                />
                <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '15px', fontSize: '0.9rem', color: '#2C3E50' }}>
                  <span>文字数: {hintText.length}</span>
                  {hintText.length > 0 && (
                    <span style={{ color: '#F39C12', fontWeight: 'bold' }}>
                      予想スコア: {18 - Math.ceil(hintText.length / (currentRoundData.hints.length + 1))}
                    </span>
                  )}
                </div>
                <button
                  onClick={submitHint}
                  disabled={!hintText.trim()}
                  style={{
                    width: '100%',
                    padding: '12px',
                    background: hintText.trim() ? '#F39C12' : '#ccc',
                    color: '#fff',
                    border: 'none',
                    borderRadius: '6px',
                    cursor: hintText.trim() ? 'pointer' : 'not-allowed',
                    fontWeight: 'bold',
                  }}
                >
                  ヒントを投稿
                </button>

                {currentRoundData.hints.length > 0 && (
                  <div style={{ marginTop: '20px', paddingTop: '20px', borderTop: '2px solid #ddd' }}>
                    <h4 style={{ color: '#2C3E50', marginBottom: '10px' }}>投稿されたヒント</h4>
                    {currentRoundData.hints.map((hint, idx) => (
                      <div
                        key={idx}
                        style={{
                          background: '#f5f5f5',
                          padding: '12px',
                          marginBottom: '10px',
                          borderRadius: '6px',
                          borderLeft: '4px solid #F39C12',
                        }}
                      >
                        <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                          <span>#{hint.order} {hint.text}</span>
                          <span style={{ color: '#7F8C8D', fontSize: '0.9rem' }}>
                            {hint.charCount}字 → {hint.score}点
                          </span>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}

            {isParent && currentRoundData?.status === 'hint_phase' && (
              <div>
                <h2 style={{ color: '#E74C3C', marginBottom: '15px' }}>👑 ヒント出題中...</h2>
                <p style={{ color: '#666', marginBottom: '15px' }}>子たちのヒントを見ています</p>
                {currentRoundData.hints.length > 0 && (
                  <div>
                    <h4 style={{ color: '#2C3E50', marginBottom: '10px' }}>現在のヒント:</h4>
                    {currentRoundData.hints.map((hint, idx) => (
                      <div
                        key={idx}
                        style={{
                          background: 'linear-gradient(135deg, rgba(243, 156, 18, 0.1), rgba(231, 76, 60, 0.1))',
                          padding: '15px',
                          marginBottom: '10px',
                          borderRadius: '6px',
                          borderLeft: '4px solid #F39C12',
                        }}
                      >
                        <div style={{ fontSize: '1.1rem', fontWeight: 'bold', marginBottom: '5px', color: '#2C3E50' }}>
                          {hint.text}
                        </div>
                        <div style={{ color: '#7F8C8D', fontSize: '0.9rem' }}>
                          {hint.charCount}字
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}

            {currentRoundData?.status === 'finished' && (
              <div>
                <h2 style={{ color: '#E74C3C', marginBottom: '15px' }}>ラウンド終了</h2>
                <p style={{ fontSize: '1.1rem', marginBottom: '10px', color: '#2C3E50' }}>
                  正解: <strong>{currentRoundData?.answer}</strong>
                </p>
                <p style={{ fontSize: '1.3rem', marginBottom: '20px' }}>
                  {currentRoundData?.correctAnswer ? '✅ 正解！' : '❌ ハズレ'}
                </p>
                {currentRound < game.rounds.length - 1 ? (
                  <button
                    onClick={() => {
                      setCurrentRound(currentRound + 1);
                      setAnswer('');
                      setHintText('');
                    }}
                    style={{
                      width: '100%',
                      padding: '12px',
                      background: '#E74C3C',
                      color: '#fff',
                      border: 'none',
                      borderRadius: '6px',
                      cursor: 'pointer',
                      fontWeight: 'bold',
                    }}
                  >
                    次のラウンドへ
                  </button>
                ) : (
                  <button
                    onClick={() => setAppState('result')}
                    style={{
                      width: '100%',
                      padding: '12px',
                      background: '#E74C3C',
                      color: '#fff',
                      border: 'none',
                      borderRadius: '6px',
                      cursor: 'pointer',
                      fontWeight: 'bold',
                    }}
                  >
                    最終結果を見る
                  </button>
                )}
              </div>
            )}

            {errorMsg && (
              <div style={{
                background: 'rgba(231, 76, 60, 0.1)',
                border: '2px solid #E74C3C',
                color: '#E74C3C',
                padding: '12px',
                borderRadius: '6px',
                marginTop: '15px',
                fontWeight: 'bold',
              }}>
                {errorMsg}
              </div>
            )}
          </div>
        </div>
      </div>
    );
  }

  // 結果画面
  if (appState === 'result' && game) {
    const sorted = [...game.players].sort((a, b) => b.totalScore - a.totalScore);
    return (
      <div style={{ maxWidth: '600px', margin: '0 auto', padding: '20px' }}>
        <h1 style={{ textAlign: 'center', fontSize: '2.5rem', color: '#E74C3C', marginBottom: '30px' }}>
          ゲーム終了！
        </h1>

        <div style={{ background: '#fff', padding: '30px', borderRadius: '12px', boxShadow: '0 10px 40px rgba(0,0,0,0.1)', marginBottom: '20px' }}>
          <h2 style={{ textAlign: 'center', marginBottom: '20px', color: '#2C3E50' }}>最終順位</h2>
          {sorted.map((player, idx) => (
            <div
              key={player.playerId}
              style={{
                display: 'flex',
                alignItems: 'center',
                padding: '15px',
                background: '#f5f5f5',
                marginBottom: '10px',
                borderRadius: '6px',
              }}
            >
              <span style={{ fontSize: '2rem', marginRight: '15px', minWidth: '40px' }}>
                {idx === 0 && '🥇'}
                {idx === 1 && '🥈'}
                {idx === 2 && '🥉'}
                {idx >= 3 && `${idx + 1}位`}
              </span>
              <span style={{ flex: 1, fontWeight: 'bold', color: '#2C3E50' }}>{player.name}</span>
              <span style={{ fontSize: '1.2rem', fontWeight: 'bold', color: '#E74C3C' }}>
                {player.totalScore}点
              </span>
            </div>
          ))}
        </div>

        <button
          onClick={() => {
            setAppState('lobby');
            setGameID(null);
            setPlayerID(null);
            setGame(null);
            setParentName('');
            setJoinGameID('');
            setPlayerName('');
          }}
          style={{
            width: '100%',
            padding: '12px',
            background: '#E74C3C',
            color: '#fff',
            border: 'none',
            borderRadius: '6px',
            fontSize: '1.1rem',
            fontWeight: 'bold',
            cursor: 'pointer',
          }}
        >
          ロビーに戻る
        </button>
      </div>
    );
  }

  return <div style={{ padding: '20px', textAlign: 'center' }}>読み込み中...</div>;
};

export default DenpouApp;
