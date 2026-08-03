import {create} from '@bufbuild/protobuf';
import {
  LogLevels as LogLevelsProto,
  LogLevelsSchema,
  LogLevels_LogLevel as LogLevelProto,
  LogLevels_LogLevelSchema,
  LogLevels_Level as LevelProto,
} from '../../../../api/config/v1/resources_pb';
import {isNumeric} from '../../func';

export enum Level {
  UNDEFINED = 0,
  ALL = 1,
  TRACE,
  DEBUG,
  INFO,
  WARN,
  ERROR,
  FATAL,
  OFF
}

export const levels: Level[] = Object.keys(Level)
  .filter(_ => !isNumeric(_))
  .filter(_ => Level[_] !== Level.UNDEFINED)
  .map(key => Level[key]);

export class LogLevel {
  logger: string;
  level: Level;

  constructor({logger = '', level = Level.UNDEFINED}: Partial<LogLevel> = {}) {
    this.logger = logger;
    this.level = level;
  }

  static toProto(logLevel: LogLevel): LogLevelProto {
    return create(LogLevels_LogLevelSchema, {
      logger: logLevel.logger,
      level: logLevel.level as unknown as LevelProto
    });
  }

  static fromProto(logLevel: LogLevelProto): LogLevel {
    return new LogLevel({
      level: logLevel.level as unknown as Level,
      logger: logLevel.logger
    });
  }
}

export class LogLevels {
  logLevelList: LogLevel[];

  constructor({logLevelList = []}: Partial<LogLevels> = {}) {
    this.logLevelList = logLevelList;
  }

  static toProto(logLevels: LogLevels): LogLevelsProto {
    return create(LogLevelsSchema, {logLevel: logLevels.logLevelList.map(LogLevel.toProto)});
  }

  static fromProto(logLevels: LogLevelsProto): LogLevels {
    return {
      logLevelList: logLevels.logLevel.map(LogLevel.fromProto)
    };
  }
}
