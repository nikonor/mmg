use clap::Parser;
use csv::{ReaderBuilder, WriterBuilder};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::fs;
use std::io;
use std::path::PathBuf;
use std::process;

/// Программа для жеребьёвки турнира по Го (система Мак-Магон)
#[derive(Parser, Debug)]
#[command(author, version, about, long_about = None)]
struct Args {
    /// Номер следующего раунда (например, 2 для второго раунда)
    #[arg(short, long)]
    next_round: usize,

    /// Базовое имя файла (без расширения и номера тура)
    #[arg(short, long)]
    base: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
struct Player {
    name: String,
    rating: i32,
    mcmahon_group: Option<usize>,
    points: Option<f32>,
    place: Option<usize>,
    berger_coeff: Option<f32>,
    buchholz_coeff: Option<f32>,
    round_results: Vec<String>,
    /// Стабильный ID игрока (place из входного файла, не меняется при сортировке)
    original_place: usize,
}

#[derive(Debug, Clone)]
struct Tournament {
    players: Vec<Player>,
    current_round: usize,
}

/// Генерирует путь к входному файлу по номеру тура
/// Тур 1 -> base.csv
/// Тур 2 -> base.001.csv
/// Тур 3 -> base.002.csv
fn generate_input_path(base_name: &str, round: usize) -> PathBuf {
    if round == 1 {
        PathBuf::from(format!("{}.csv", base_name))
    } else {
        PathBuf::from(format!("{}.{}.csv", base_name, format!("{:03}", round - 1)))
    }
}

/// Генерирует путь к выходному файлу по номеру тура
/// Тур 1 -> base.001.csv
/// Тур 2 -> base.002.csv
fn generate_output_path(base_name: &str, next_round: usize) -> PathBuf {
    PathBuf::from(format!(
        "{}.{}.csv",
        base_name,
        format!("{:03}", next_round)
    ))
}

impl Player {
    fn new(name: String, rating: i32, original_place: usize) -> Self {
        Self {
            name,
            rating,
            mcmahon_group: None,
            points: None,
            place: None,
            berger_coeff: None,
            buchholz_coeff: None,
            round_results: Vec::new(),
            original_place,
        }
    }

    /// Получить номер противника из результата (для обратной совместимости)
    /// Формат: "номер:" (победа), "номер:-" (поражение), "номер:=" (ничья)
    fn parse_opponent(result: &str) -> Option<usize> {
        if result.is_empty() || result == "X" {
            return None;
        }
        let colon_pos = result.find(':')?;
        let before_colon = &result[..colon_pos];
        if before_colon.is_empty() {
            return None;
        }
        before_colon.parse().ok()
    }

    /// Проверить, был ли это победа (заканчивается на ':' без дополнительных символов)
    fn is_win(result: &str) -> bool {
        result.ends_with(':') && !result.ends_with(":-") && !result.ends_with(":=")
    }
}

impl Tournament {
    fn from_csv(path: &PathBuf) -> Result<Self, io::Error> {
        let content = fs::read_to_string(path)?;
        let mut rdr = ReaderBuilder::new()
            .has_headers(false)
            .from_reader(content.as_bytes());

        let mut players = Vec::new();

        for result in rdr.records() {
            let record = result.map_err(|e| io::Error::new(io::ErrorKind::Other, e))?;

            if record.len() < 8 {
                eprintln!("Неверный формат строки (минимум 8 полей): {:?}", record);
                continue;
            }

            // Формат: place,original_place,Имя,рейтинг,группа,очки,Бергер,Бухгольц,результаты...
            let place: Option<usize> = record[0].trim().parse().ok();
            let original_place: usize = record[1].trim().parse().unwrap_or_else(|_| {
                eprintln!("Ошибка парсинга original_place для строки: {:?}", record);
                0
            });
            let name = record[2].trim().to_string();
            let rating: i32 = record[3].trim().parse().unwrap_or_else(|_| {
                eprintln!("Ошибка парсинга рейтинга для {}: {}", name, &record[3]);
                0
            });
            let mcmahon_group: Option<usize> = record[4].trim().parse().ok();
            let points: Option<f32> = record[5].trim().parse().ok();
            let berger_coeff: Option<f32> = record[6].trim().parse().ok();
            let buchholz_coeff: Option<f32> = record[7].trim().parse().ok();

            let mut player = Player::new(name, rating, original_place);
            player.place = place;
            player.mcmahon_group = mcmahon_group;
            player.points = points;
            player.berger_coeff = berger_coeff;
            player.buchholz_coeff = buchholz_coeff;

            // Парсим результаты раундов
            for i in 8..record.len() {
                let result = record[i].trim().to_string();
                if !result.is_empty() {
                    player.round_results.push(result);
                }
            }

            players.push(player);
        }

        // Определяем текущий раунд по количеству результатов
        let current_round = if players.is_empty() {
            1
        } else {
            players[0].round_results.len() + 1
        };

        Ok(Tournament {
            players,
            current_round,
        })
    }

    /// Рассчитать коэффициенты Бухгольца и Бергера
    fn calculate_coefficients(&mut self) {
        // Маппинг: original_place (стабильный ID) -> очки игрока
        let id_to_points: HashMap<usize, f32> = self
            .players
            .iter()
            .map(|p| (p.original_place, p.points.unwrap_or(0.0)))
            .collect();

        for i in 0..self.players.len() {
            let mut buchholz = 0.0;
            let mut berger = 0.0;

            for result in &self.players[i].round_results {
                if let Some(opp_id) = Player::parse_opponent(result) {
                    let opp_points = id_to_points.get(&opp_id).copied().unwrap_or(0.0);
                    buchholz += opp_points;
                    if Player::is_win(result) {
                        berger += opp_points;
                    }
                }
            }

            self.players[i].buchholz_coeff = Some(buchholz);
            self.players[i].berger_coeff = Some(berger);
        }
    }

    /// Сортировать игроков по: очки → Бухгольц → Бергер
    fn sort_players(&mut self) {
        self.players.sort_by(|a, b| {
            // По очкам (убывание)
            let points_cmp = b
                .points
                .unwrap_or(0.0)
                .partial_cmp(&a.points.unwrap_or(0.0))
                .unwrap_or(std::cmp::Ordering::Equal);
            if points_cmp != std::cmp::Ordering::Equal {
                return points_cmp;
            }

            // По Бухгольцу (убывание)
            let buchholz_cmp = b
                .buchholz_coeff
                .unwrap_or(0.0)
                .partial_cmp(&a.buchholz_coeff.unwrap_or(0.0))
                .unwrap_or(std::cmp::Ordering::Equal);
            if buchholz_cmp != std::cmp::Ordering::Equal {
                return buchholz_cmp;
            }

            // По Бергеру (убывание)
            b.berger_coeff
                .unwrap_or(0.0)
                .partial_cmp(&a.berger_coeff.unwrap_or(0.0))
                .unwrap_or(std::cmp::Ordering::Equal)
        });
    }

    /// Проверить, играли ли уже два игрока между собой
    fn have_played_each_other(&self, idx1: usize, idx2: usize) -> bool {
        // original_place — стабильный ID, не меняется при сортировке
        let opp_original = self.players[idx2].original_place;
        self.players[idx1]
            .round_results
            .iter()
            .any(|r| Player::parse_opponent(r) == Some(opp_original))
    }

    /// Создать пару игроков — записываем original_place соперника (стабильный ID)
    fn make_pair(&mut self, idx1: usize, idx2: usize) {
        let p1_opp = self.players[idx2].original_place;
        let p2_opp = self.players[idx1].original_place;

        self.players[idx1]
            .round_results
            .push(format!("{}:", p1_opp));
        self.players[idx2]
            .round_results
            .push(format!("{}:", p2_opp));
    }

    /// Создать пары для следующего раунда
    fn create_pairs(&mut self) {
        let n = self.players.len();
        let mut paired = vec![false; n];

        // assign_places уже вызван и проставил place по текущей позиции
        // Жеребьёвка: идём сверху вниз, ищем пару с минимальной разницей очков

        // Первый проход: без повторов
        for i in 0..n {
            if paired[i] {
                continue;
            }

            let pts_i = self.players[i].points.unwrap_or(0.0);

            let mut best_j: Option<usize> = None;
            let mut best_diff = f32::MAX;

            for j in (i + 1)..n {
                if paired[j] || self.have_played_each_other(i, j) {
                    continue;
                }

                let pts_j = self.players[j].points.unwrap_or(0.0);
                let diff = (pts_i - pts_j).abs();

                if diff < best_diff {
                    best_diff = diff;
                    best_j = Some(j);
                }
            }

            if let Some(j) = best_j {
                self.make_pair(i, j);
                paired[i] = true;
                paired[j] = true;
            }
        }

        // Второй проход: если остались неспаренные, разрешаем повторы
        let unpaired: Vec<usize> = (0..n).filter(|&i| !paired[i]).collect();

        if unpaired.len() > 1 {
            for i in (0..unpaired.len()).step_by(2) {
                if i + 1 < unpaired.len() {
                    let p1 = unpaired[i];
                    let p2 = unpaired[i + 1];
                    self.make_pair(p1, p2);
                    paired[p1] = true;
                    paired[p2] = true;
                }
            }
        }

        // Если остался один без пары - это последний в таблице (bye)
        let unpaired: Vec<usize> = (0..n).filter(|&i| !paired[i]).collect();
        if unpaired.len() == 1 {
            let last = unpaired[0];
            if last == n - 1 {
                self.players[last].round_results.push("X".to_string());
                paired[last] = true;
            } else {
                eprintln!(
                    "Предупреждение: игрок {} (не последний) остаётся без пары!",
                    self.players[last].name
                );
            }
        }
    }

    fn assign_places(&mut self) {
        for (idx, player) in self.players.iter_mut().enumerate() {
            player.place = Some(idx + 1);
        }
    }

    fn save_to_csv(&self, path: &PathBuf) -> Result<(), io::Error> {
        let mut wtr = WriterBuilder::new().flexible(true).from_path(path)?;

        for player in &self.players {
            let mut record: Vec<String> = vec![
                player.place.map(|p| p.to_string()).unwrap_or_default(),
                player.original_place.to_string(),
                player.name.clone(),
                player.rating.to_string(),
                player
                    .mcmahon_group
                    .map(|g| g.to_string())
                    .unwrap_or_default(),
                player
                    .points
                    .map(|p| format!("{:.1}", p))
                    .unwrap_or_default(),
                player
                    .berger_coeff
                    .map(|c| format!("{:.1}", c))
                    .unwrap_or_default(),
                player
                    .buchholz_coeff
                    .map(|c| format!("{:.1}", c))
                    .unwrap_or_default(),
            ];

            for result in &player.round_results {
                record.push(result.clone());
            }

            wtr.write_record(&record)
                .map_err(|e| io::Error::new(io::ErrorKind::Other, e))?;
        }

        wtr.flush()
            .map_err(|e| io::Error::new(io::ErrorKind::Other, e))?;
        Ok(())
    }
}

fn run(args: Args) -> Result<(), io::Error> {
    let next_round = args.next_round;

    if next_round < 2 {
        eprintln!(
            "Номер раунда должен быть >= 2. Для первого раунда используйте отдельную программу."
        );
        process::exit(1);
    }

    let current_round = next_round - 1;

    // Определяем путь к входному файлу
    let input_path = generate_input_path(&args.base, next_round);

    // Проверяем, что файл предыдущего тура существует
    if !input_path.exists() {
        eprintln!("Файл предыдущего тура не найден: {:?}", input_path);
        eprintln!("Ожидается файл с результатами раунда {}", current_round);
        process::exit(1);
    }

    println!("Загрузка турнира из файла: {:?}", input_path);
    let mut tournament = Tournament::from_csv(&input_path)?;

    // Проверяем, что результаты предыдущего тура есть
    if tournament.current_round < next_round {
        eprintln!(
            "Недостаточно результатов. Ожидается раунд {}, имеется {} раундов.",
            next_round,
            tournament.current_round - 1
        );
        process::exit(1);
    }

    println!("Рассчитываю коэффициенты...");
    tournament.calculate_coefficients();

    println!("Сортирую игроков...");
    tournament.sort_players();

    // Выводим отсортированных игроков для отладки
    println!("Отсортированные игроки:");
    for (i, p) in tournament.players.iter().enumerate() {
        println!(
            "  {}. {} (очки: {:.1}, Бухгольц: {:.1}, Бергер: {:.1})",
            i + 1,
            p.name,
            p.points.unwrap_or(0.0),
            p.buchholz_coeff.unwrap_or(0.0),
            p.berger_coeff.unwrap_or(0.0)
        );
    }

    println!("Создаю пары для раунда {}...", next_round);
    tournament.create_pairs();

    // Обновляем place для выходного файла (позиция в таблице)
    tournament.assign_places();

    // Сохраняем результаты
    let output_path = generate_output_path(&args.base, next_round);
    tournament.save_to_csv(&output_path)?;
    println!("Результаты сохранены в: {:?}", output_path);

    // Выводим пары
    println!("Пары на раунд {}:", next_round);
    for p in &tournament.players {
        if let Some(last_result) = p.round_results.last() {
            if let Some(opp_id) = Player::parse_opponent(last_result) {
                // Ищем игрока по original_place
                let opp_name = tournament
                    .players
                    .iter()
                    .find(|pp| pp.original_place == opp_id)
                    .map(|pp| pp.name.as_str())
                    .unwrap_or("?");
                println!("  {} vs {}", p.name, opp_name);
            } else if last_result == "X" {
                println!("  {} - пропуск (bye)", p.name);
            }
        }
    }

    Ok(())
}

fn main() {
    let args = Args::parse();

    if let Err(e) = run(args) {
        eprintln!("Ошибка: {}", e);
        process::exit(1);
    }
}
