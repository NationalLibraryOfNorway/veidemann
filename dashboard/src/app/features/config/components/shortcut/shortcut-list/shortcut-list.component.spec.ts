import {ComponentFixture, TestBed} from '@angular/core/testing';

import {ShortcutListComponent} from './shortcut-list.component';
import {ConfigObject, Kind} from '../../../../shared/models';
import {CoreTestingModule} from '../../../../core/core.testing.module';
import {CommonsModule} from '../../../../commons';
import {EntityNamePipe} from '../../../pipe';
import {ConfigService} from '../../../../commons/services';
import {of} from 'rxjs';
import {RouterTestingModule} from '@angular/router/testing';

describe('ShortcutListComponent', () => {
  let component: ShortcutListComponent;
  let fixture: ComponentFixture<ShortcutListComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      declarations: [ShortcutListComponent, EntityNamePipe],
      imports: [CoreTestingModule.forRoot(), RouterTestingModule, CommonsModule],
      providers: [
        {
          provide: ConfigService, useValue: {
            get: () => of(null)
          }
        }
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(ShortcutListComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should create with SeedConfig', async () => {
    component.configObject = new ConfigObject({kind: Kind.SEED});
    await fixture.whenStable();
    expect(component).toBeTruthy();
  });

  it('should create with CrawlConfig', async () => {
    component.configObject = new ConfigObject({kind: Kind.CRAWLCONFIG});
    await fixture.whenStable();
    expect(component).toBeTruthy();
  });
});
